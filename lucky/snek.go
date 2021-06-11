package lucky

import (
	"crypto/rand"
	"fmt"
	"log"

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
	if s.currentState != nil || len(s.previousStates) > 0 {
		return fmt.Errorf("cannot start a game in progress")
	}
	s.currentState = st
	s.previousStates = nil
	return nil
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
	// Where can I go?
	directions := s.affordances()
	// Decide on a direction
	var b [1]byte
	if _, err := rand.Reader.Read(b[:]); err != nil {
		log.Fatalf("failed to read random bytes: %v", err)
	}
	i := int(b[0]) % len(directions)
	return directions[i]
}

func (s *snek) affordances() []string {
	head := s.currentState.Me.Head
	blocked := map[string]bool{}
	// Check if walls are blocking some moves
	if head.X == 0 {
		blocked["left"] = true
	}
	if head.X == s.currentState.Board.Width {
		blocked["right"] = true
	}
	if head.Y == 0 {
		blocked["down"] = true
	}
	if head.Y == s.currentState.Board.Height {
		blocked["up"] = true
	}
	// A primitive and weak sense organ
	sense := func(point api.Point, avoid, grab bool) (string, bool) {
		var adjacency string
		if point == head {
			return "", false
		}
		if point.X == head.X {
			if point.Y == head.Y+1 {
				adjacency = "up"
			}
			if point.Y == head.Y-1 {
				adjacency = "down"
			}
		}
		if point.Y == head.Y {
			if point.X == head.X+1 {
				adjacency = "right"
			}
			if point.X == head.X-1 {
				adjacency = "left"
			}
		}
		if avoid {
			blocked[adjacency] = true
			return "", false
		}
		if grab {
			return adjacency, true
		}
		return "", false
	}
	// Go for food
	for _, point := range s.currentState.Board.Food {
		if direction, hit := sense(point, false, true); hit {
			return []string{direction}
		}
	}
	// Go for the head
	var heads []api.Point
	for _, snake := range s.currentState.Board.Snakes {
		heads = append(heads, snake.Head)
		for _, point := range snake.Body {
			sense(point, true, false)
		}
	}
	// Avoid self collision
	for _, point := range s.currentState.Me.Body {
		sense(point, true, false)
	}
	// Avoid peer collision
	for _, point := range heads {
		if direction, hit := sense(point, false, true); hit {
			return []string{direction}
		}
	}
	// Anything we're not avoiding is an affordance
	var affordances []string
	for _, dir := range []string{"up", "down", "left", "right"} {
		if _, ok := blocked[dir]; !ok {
			affordances = append(affordances, dir)
		}
	}
	return affordances
}

func (s *snek) End(st *api.State) error {
	if s.currentState == nil {
		return fmt.Errorf("game not started")
	}
	s.previousStates = append(s.previousStates, s.currentState, st)
	s.currentState = nil
	return nil
}
