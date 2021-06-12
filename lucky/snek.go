// Package lucky contains a Battlesnake with no sense of time or prediction.
// This snake reacts only to the current board state on each move, sensing the board
// by plotting a graph of inhabitable spaces from current degrees of freedom, to a max
// depth. This max depth simulates the limits of lucky's sense organs; in practice,
// it is bounded by the CPU required by this naive brute-force approach.
//
// Degrees of freedom are spaces not occupied by walls or snakes (including self),
// or free spaces with less degrees of freedom than current snake length.
//
// lucky's decision-making is a result of several drives, in descending order
// of precedence:
// - Survival (avoid walls, snakes, and obvious tight spaces)
// - Strike prey if immediately adjacent
// - Seek food
// - Seek freedom, if there is a strongly discernable difference. Otherwise,
//   choose a random direction.
//
// lucky is named for the initial implementation, which used a purely random
// selection of non-ending moves.  This quickly proved ineffective, but the
// randomness lives on in the random walk used when sensing the board, and when
// a direction is otherwise unclear. Earlier versions used randomness as a
// placeholder for any sort of decision making.
//
// Limitations:
// - The lack of temporal awareness means lucky avoids tight spaces where the
//   tail will clear up before a crash. This inhibits coiling behavior, which
//   would be advantageous at longer lengths.
// - The brute force approach to sensing is not very efficient, and needs
//   optimization.
// - Failure at snake avoidance. Even if the "strike immediately" behavior is
//   inhibited, lucky will still eat smaller snakes "accidentally" without
//   temporality and prediction.
// - Hazards are sensed, but not acted on yet.
//
package lucky

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/cmars/snekfarm/api"
)

// New returns a new lucky api.Snek.
func New() api.Snek {
	return &snek{}
}

type snek struct {
	currentState *api.State
	board        board
}

type board struct {
	w, h int
	m    map[api.Point]string
}

func newBoard(w, h int) *board {
	return &board{w: w, h: h, m: map[api.Point]string{}}
}

func (b board) set(p api.Point, val string) {
	b.m[p] = val
}

func (b board) get(p api.Point) string {
	if p.X <= -1 {
		return "wall"
	}
	if p.X >= b.w {
		return "wall"
	}
	if p.Y <= -1 {
		return "wall"
	}
	if p.Y >= b.h {
		return "wall"
	}
	return b.m[p]
}

func (s *snek) Start(st *api.State) error {
	// Reseed the PRNG just to shake off any residual determinism. Probably
	// useless but also harmless.
	s.reseed()
	if s.currentState != nil {
		return fmt.Errorf("cannot start a game in progress")
	}
	s.currentState = st
	return nil
}

func (s *snek) reseed() {
	var b [8]byte
	var seed int64
	_, err := crand.Reader.Read(b[:])
	if err != nil {
		seed = time.Now().UTC().UnixNano()
	} else {
		seed, _ = binary.Varint(b[:])
	}
	rand.Seed(seed)
}

func (s *snek) Move(st *api.State) (string, string, error) {
	if s.currentState == nil {
		return "", "", fmt.Errorf("game not started")
	}
	s.currentState = st
	direction := s.direction()
	return direction, "", nil
}

func (s *snek) direction() string {
	b := s.observe()
	origin := orient(b, s.currentState.Me.Head)
	return decide(origin, s.currentState.Me.Length)
}

func (s *snek) observe() *board {
	b := newBoard(s.currentState.Board.Width, s.currentState.Board.Height)
	for _, snake := range s.currentState.Board.Snakes {
		for i, point := range snake.Body {
			if i == 0 && snake.Length < s.currentState.Me.Length {
				b.set(point, "prey")
			} else {
				b.set(point, "snek")
			}
		}
	}
	for _, point := range s.currentState.Board.Food {
		b.set(point, "food")
	}
	for _, point := range s.currentState.Board.Hazards {
		b.set(point, "hazard")
	}
	return b
}

const maxOrientDepth = 12

func orient(b *board, head api.Point) *node {
	origin := &node{Point: head}
	var cur *node
	next := []*node{origin}
	visited := map[api.Point]int{}
	for len(next) > 0 {
		cur, next = next[0], next[1:]
		if len(cur.parents) == maxOrientDepth {
			continue
		}
		parents := append([]*node{cur}, cur.parents...)
		// Check adjacent positions.
		var adjacent []*node
		up, down, left, right := cur.above(), cur.below(), cur.leftOf(), cur.rightOf()
		if visited[up] < 2 { // Allow sensing in adjacent directions to overlap
			if n := newNode(up, b.get(up), parents...); n != nil {
				cur.up = n
				if len(parents) < maxOrientDepth {
					adjacent = append(adjacent, n)
				}
			}
		}
		if visited[down] < 2 {
			if n := newNode(down, b.get(down), parents...); n != nil {
				cur.down = n
				if len(parents) < maxOrientDepth {
					adjacent = append(adjacent, n)
				}
			}
		}
		if visited[left] < 2 {
			if n := newNode(left, b.get(left), parents...); n != nil {
				cur.left = n
				if len(parents) < maxOrientDepth {
					adjacent = append(adjacent, n)
				}
			}
		}
		if visited[right] < 2 {
			if n := newNode(right, b.get(right), parents...); n != nil {
				cur.right = n
				if len(parents) < maxOrientDepth {
					adjacent = append(adjacent, n)
				}
			}
		}
		visited[cur.Point]++
		// Randomize the walk so we get a more uniform spread out from the
		// origin. Otherwise the graphs will favor one direction over the
		// others.
		rand.Shuffle(len(adjacent), func(i, j int) {
			adjacent[i], adjacent[j] = adjacent[j], adjacent[i]
		})
		next = append(next, adjacent...)
	}
	return origin
}

func decide(origin *node, length int) string {
	var ms []*move
	// Add possible moves that we can fit into. This doesn't account for
	// growth in a tight space, or a retreating tail.
	if origin.up != nil {
		if m := newMove(origin.up, "up"); m != nil && m.freedom > length {
			ms = append(ms, m)
		}
	}
	if origin.down != nil {
		if m := newMove(origin.down, "down"); m != nil && m.freedom > length {
			ms = append(ms, m)
		}
	}
	if origin.left != nil {
		if m := newMove(origin.left, "left"); m != nil && m.freedom > length {
			ms = append(ms, m)
		}
	}
	if origin.right != nil {
		if m := newMove(origin.right, "right"); m != nil && m.freedom > length {
			ms = append(ms, m)
		}
	}
	if len(ms) == 0 {
		log.Println("out of moves!")
		return ""
	}
	var maxYum, maxYumAt int
	maxFree, maxFreeAt, minFree := 0, 0, math.MaxInt32
	for i := range ms {
		if ms[i].canStrike {
			return ms[i].direction
		}
		if ms[i].yum > maxYum {
			maxYum, maxYumAt = ms[i].yum, i
		}
		if ms[i].freedom > maxFree {
			maxFree, maxFreeAt = ms[i].freedom, i
		}
		if ms[i].freedom < minFree {
			minFree = ms[i].freedom
		}
	}
	if maxYum < yumScentRange {
		// Without a strong food signal, aim for a clearly more open space.
		if maxFree-minFree > length/2 {
			return ms[maxFreeAt].direction
		}
		// Otherwise, mix it up, so we don't get stuck in circles.
		sort.Sort(jitterMoves(ms))
		return ms[len(ms)-1].direction
	}
	// Prefer the yummiest direction
	return ms[maxYumAt].direction
}

type move struct {
	*node
	direction string
}

func newMove(n *node, dir string) *move {
	return &move{
		node:      n,
		direction: dir,
	}
}

type jitterMoves []*move

func (m jitterMoves) Len() int { return len(m) }
func (m jitterMoves) Less(i, j int) bool {
	// Wander, but not into trouble if we can help it
	if m[i].yuck > m[j].yuck {
		return true
	}
	return rand.Int()%2 == 0
}
func (m jitterMoves) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

type node struct {
	api.Point
	parents []*node

	// Where can we go from here?
	up    *node
	down  *node
	left  *node
	right *node

	// What is good in this life?
	// To gobble up all the foods
	yum int // cumulative
	// to strectch out free in open places
	freedom int // cumulative
	// to stay out of the muck
	yuck int // cumulative
	// to strike at the heads of my enemies
	canStrike bool
}

// Food is worth treking across the board for, use a longer range.
const yumScentRange = 10

// Hazards are only worth paying attention to at a short range. If it's not
// nearby, it probably doesn't need to factor.
const yuckScentRange = 2

func newNode(p api.Point, val string, parents ...*node) *node {
	if val == "wall" || val == "snek" {
		return nil
	}
	n := &node{Point: p, parents: parents}
	if val == "food" {
		n.freedom++
		// TODO: add a hunger factor
		if len(parents) < yumScentRange {
			attraction := yumScentRange - len(parents)
			n.yum = n.yum + attraction
			for i := range parents {
				if i < attraction {
					parents[i].yum = parents[i].yum + attraction - i
				}
				n.freedom++
			}
		}
	}
	if val == "hazard" {
		// TODO: improve yuck avoidance?
		n.freedom++ // yucky but still free
		if len(parents) < yuckScentRange {
			repulsion := yuckScentRange - len(parents)
			n.yuck = n.yuck + repulsion
			for i := range parents {
				if i >= repulsion {
					break
				}
				parents[i].yuck = parents[i].yuck + repulsion - i
			}
		}
	}
	if val == "" {
		n.freedom++
		for i := range parents {
			parents[i].freedom++
		}
	}
	if val == "prey" && len(parents) == 1 {
		n.canStrike = true
	}
	return n
}

func (n *node) above() api.Point {
	return api.Point{X: n.Point.X, Y: n.Point.Y + 1}
}
func (n *node) below() api.Point {
	return api.Point{X: n.Point.X, Y: n.Point.Y - 1}
}
func (n *node) leftOf() api.Point {
	return api.Point{X: n.Point.X - 1, Y: n.Point.Y}
}
func (n *node) rightOf() api.Point {
	return api.Point{X: n.Point.X + 1, Y: n.Point.Y}
}

func (s *snek) End(st *api.State) error {
	if s.currentState == nil {
		return nil
	}
	s.currentState = nil
	return nil
}
