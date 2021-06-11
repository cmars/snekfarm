package lucky

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/cmars/snekfarm/api"
)

func New() api.Snek {
	return &snek{}
}

type snek struct {
	currentState   *api.State
	previousStates []*api.State
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
	// Where can I go?
	directions := s.affordances()
	if len(directions) == 0 {
		return ""
	}
	// Decide on a direction
	i := rand.Intn(len(directions))
	log.Printf("affordances: %#v choose: %v", directions, directions[i])
	return directions[i]
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
