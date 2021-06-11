package lucky

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/cmars/snekfarm/api"
)

func New() api.Snek {
	return &snek{}
}

type snek struct {
	currentState   *api.State
	previousStates []*api.State
	board          board
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

	log.Printf("%#v", s.currentState)
	direction := s.direction()
	return direction, "", nil
}

func (s *snek) direction() string {
	b := s.observe()
	origin := s.orient(b)
	return s.decide(origin)
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

func (s *snek) orient(b *board) *node {
	origin := &node{Point: s.currentState.Me.Head}
	var cur *node
	next := []*node{origin}
	for len(next) > 0 {
		cur, next = next[0], next[1:]
		parents := append([]*node{cur}, cur.parents...)
		up, down, left, right := cur.above(), cur.below(), cur.leftOf(), cur.rightOf()
		if n := newNode(up, b.get(up), parents...); n != nil {
			cur.up = n
			next = append(next, n)
		}
		if n := newNode(down, b.get(down), parents...); n != nil {
			cur.down = n
			next = append(next, n)
		}
		if n := newNode(up, b.get(left), parents...); n != nil {
			cur.left = n
			next = append(next, n)
		}
		if n := newNode(up, b.get(right), parents...); n != nil {
			cur.right = n
			next = append(next, n)
		}
	}
	return origin
}

func (s *snek) decide(origin *node) string {
	var ms moves
	if m := s.newMove(origin.up, "up"); m != nil {
		ms = append(ms, m)
	}
	if m := s.newMove(origin.down, "down"); m != nil {
		ms = append(ms, m)
	}
	if m := s.newMove(origin.left, "left"); m != nil {
		ms = append(ms, m)
	}
	if m := s.newMove(origin.right, "right"); m != nil {
		ms = append(ms, m)
	}
	sort.Sort(ms)
	if len(ms) > 0 {
		return ms[0].direction
	}
	log.Println("out of moves")
	return ""
}

type move struct {
	*node
	direction string
	heuristic float64
}

func (s *snek) newMove(n *node, dir string) *move {
	return &move{
		node:      n,
		direction: dir,
		heuristic: s.heuristic(n),
	}
}

func (s *snek) heuristic(n *node) float64 {
	// Food is best, then freedom, then the least yuck involved.
	value := float64(n.yum)*3.0 + float64(n.freedom)*2.0 - float64(n.yuck)*0.5
	// Opportunistic predation!
	if n.canStrike {
		value = value + 50.0
	}
	return value
}

type moves []*move

func (m moves) Len() int { return len(m) }
func (m moves) Less(i, j int) bool {
	return m[i].heuristic < m[j].heuristic
}
func (m moves) Swap(i, j int) {
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
		n.yum = n.yum + yumScentRange
		for i := range parents {
			if i < yumScentRange {
				// Sense of scent decays over distance
				parents[i].yum = parents[i].yum + (yumScentRange - i)
			}
			n.freedom++
		}
	}
	if val == "hazard" {
		// TODO: improve yuck avoidance?
		n.yuck = n.yuck + yuckScentRange
		for i := range parents {
			if i > yuckScentRange {
				break
			}
			parents[i].yuck = parents[i].yuck + (yuckScentRange - i)
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

func (s *snek) affordances() []string {
	head := s.currentState.Me.Head
	blocked := map[string]bool{}
	// Check if walls are blocking some moves
	if head.X == 0 {
		log.Printf("avoid left wall")
		blocked["left"] = true
	}
	if head.X == s.currentState.Board.Width-1 {
		log.Printf("avoid right wall")
		blocked["right"] = true
	}
	if head.Y == 0 {
		log.Printf("avoid bottom wall")
		blocked["down"] = true
	}
	if head.Y == s.currentState.Board.Height-1 {
		log.Printf("avoid top wall")
		blocked["up"] = true
	}
	// A primitive and weak sense organ
	sense := func(point api.Point) string {
		if point.X == head.X {
			if point.Y == head.Y+1 {
				return "up"
			}
			if point.Y == head.Y-1 {
				return "down"
			}
		}
		if point.Y == head.Y {
			if point.X == head.X+1 {
				return "right"
			}
			if point.X == head.X-1 {
				return "left"
			}
		}
		return ""
	}
	// Go for food
	for _, point := range s.currentState.Board.Food {
		if dir := sense(point); dir != "" {
			log.Printf("grab food at %#v", point)
			return []string{dir}
		}
	}
	// Deal with snakes
	for _, snake := range s.currentState.Board.Snakes {
		// Eat smaller snakes
		if snake.Length < s.currentState.Me.Length {
			if dir := sense(snake.Head); dir != "" {
				log.Printf("strike head at %#v", snake.Head)
				return []string{dir}
			}
		}
		// Avoid collision
		for _, point := range snake.Body {
			if dir := sense(point); dir != "" {
				log.Printf("avoid other snake %q", dir)
				blocked[dir] = true
			}
		}
	}
	// Anything we're not avoiding is an affordance
	var affordances []string
	for _, dir := range []string{"up", "down", "left", "right"} {
		if ok := blocked[dir]; !ok {
			affordances = append(affordances, dir)
		}
	}
	return affordances
}

func (s *snek) End(st *api.State) error {
	if s.currentState == nil {
		return nil
	}
	s.previousStates = append(s.previousStates, s.currentState, st)
	s.currentState = nil
	return nil
}
