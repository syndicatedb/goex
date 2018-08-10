package state

import (
	"errors"
	"sync"
)

type State struct {
	state map[string]string
	sync.Mutex
}

func New() *State {
	s := State{
		state: make(map[string]string),
	}
	return &s
}

func (s *State) Get(key string) (string, error) {
	s.Lock()
	defer s.Unlock()
	if s.state[key] == "" {
		return "", errors.New("Value is not found in State. Key: " + key)
	}
	return s.state[key], nil
}

func (s *State) Set(key string, value string) {
	s.Lock()
	defer s.Unlock()
	s.state[key] = value
}

func (s *State) Del(key string) {
	s.Lock()
	defer s.Unlock()
	delete(s.state, key)
}
