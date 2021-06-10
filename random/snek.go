package random

import (
	"fmt"

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
	// TODO: choose a direction to move
	return direction, "", nil
}

func (s *snek) direction() string {
	return "up"
}

func (s *snek) End(st *api.State) error {
	if s.currentState == nil {
		return fmt.Errorf("game not started")
	}
	s.previousStates = append(s.previousStates, s.currentState, st)
	s.currentState = nil
	return nil
}
