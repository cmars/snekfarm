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

func New() api.Snek {
	return &snek{}
}

func NewDocile() api.Snek {
	return &snek{docile: true}
}

type snek struct {
	currentState   *api.State
	previousStates []*api.State
	board          board
	docile         bool
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
	s.reseed()
	if s.currentState != nil || len(s.previousStates) > 0 {
		return fmt.Errorf("cannot start a game in progress")
	}
	s.currentState = st
	s.previousStates = nil
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
	s.previousStates = append(s.previousStates, s.currentState)
	s.currentState = st
	direction := s.direction()
	return direction, "", nil
}

func (s *snek) direction() string {
	b := s.observe()
	origin := orient(b, s.currentState.Me.Head)
	return decide(origin, s.currentState.Me.Length, s.docile)
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
		visited[cur.Point]++
		parents := append([]*node{cur}, cur.parents...)
		// Check adjacent positions.
		var adjacent []*node
		up, down, left, right := cur.above(), cur.below(), cur.leftOf(), cur.rightOf()
		if visited[up] < 3 {
			if n := newNode(up, b.get(up), parents...); n != nil {
				cur.up = n
				if len(parents) < maxOrientDepth {
					adjacent = append(adjacent, n)
				}
			}
		}
		if visited[down] < 3 {
			if n := newNode(down, b.get(down), parents...); n != nil {
				cur.down = n
				if len(parents) < maxOrientDepth {
					adjacent = append(adjacent, n)
				}
			}
		}
		if visited[left] < 3 {
			if n := newNode(left, b.get(left), parents...); n != nil {
				cur.left = n
				if len(parents) < maxOrientDepth {
					adjacent = append(adjacent, n)
				}
			}
		}
		if visited[right] < 3 {
			if n := newNode(right, b.get(right), parents...); n != nil {
				cur.right = n
				if len(parents) < maxOrientDepth {
					adjacent = append(adjacent, n)
				}
			}
		}
		// Randomize the walk so we get a relatively uniform spread out from
		// the origin for sensing/heuristics.
		rand.Shuffle(len(adjacent), func(i, j int) {
			adjacent[i], adjacent[j] = adjacent[j], adjacent[i]
		})
		next = append(next, adjacent...)
	}
	return origin
}

func decide(origin *node, length int, docile bool) string {
	var ms []*move
	// Add possible moves that we can fit into
	// TODO: this doesn't account for growth in a tight space
	if origin.up != nil {
		if m := newMove(origin.up, "up"); m != nil && m.freedom > length {
			if !docile || docile != m.canStrike {
				ms = append(ms, m)
			}
		}
	}
	if origin.down != nil {
		if m := newMove(origin.down, "down"); m != nil && m.freedom > length {
			if !docile || !m.canStrike {
				ms = append(ms, m)
			}
		}
	}
	if origin.left != nil {
		if m := newMove(origin.left, "left"); m != nil && m.freedom > length {
			if !docile || !m.canStrike {
				ms = append(ms, m)
			}
		}
	}
	if origin.right != nil {
		if m := newMove(origin.right, "right"); m != nil && m.freedom > length {
			if !docile || !m.canStrike {
				ms = append(ms, m)
			}
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
	// Wander, but not into trouble
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
	s.previousStates = append(s.previousStates, s.currentState, st)
	s.currentState = nil
	return nil
}
