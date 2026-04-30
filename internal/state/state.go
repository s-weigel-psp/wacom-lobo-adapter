package state

import "sync"

// State holds the current in-process tablet mapping state.
// The message loop is single-threaded, but the mutex is included
// for safety if goroutines are added later.
type State struct {
	mu     sync.Mutex
	Mapped bool
	X      int
	Y      int
	Width  int
	Height int
}

// Set records a new mapping. Called when set_mapping command succeeds.
func (s *State) Set(x, y, w, h int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Mapped = true
	s.X, s.Y, s.Width, s.Height = x, y, w, h
}

// Reset clears the mapping. Called when reset_mapping command succeeds.
func (s *State) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Mapped = false
	s.X, s.Y, s.Width, s.Height = 0, 0, 0, 0
}

// StatusResponse returns the get_status response body (D-08).
// "monitor" is reserved/null — MULTI-01 is deferred to v2.
func (s *State) StatusResponse() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]interface{}{
		"mapped":  s.Mapped,
		"x":       s.X,
		"y":       s.Y,
		"width":   s.Width,
		"height":  s.Height,
		"monitor": nil,
	}
}
