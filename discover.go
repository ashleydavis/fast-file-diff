package main

import "sync"

type side int

const (
	sideLeft side = iota
	sideRight
)

// discoveredSet tracks which relative paths have been seen on left and right.
// When the same path is seen on both sides, Add returns true (form a pair).
type discoveredSet struct {
	mu    sync.Mutex
	pool  *pathPool
	left  map[string]bool
	right map[string]bool
}

func newDiscoveredSet(pool *pathPool) *discoveredSet {
	return &discoveredSet{
		pool:  pool,
		left:  make(map[string]bool),
		right: make(map[string]bool),
	}
}

// Add records that rel was seen on the given side. It returns true when this
// completes a pair (the other side had already been seen for rel).
func (s *discoveredSet) Add(rel string, sd side) bool {
	rel = s.pool.Intern(rel)
	s.mu.Lock()
	defer s.mu.Unlock()
	switch sd {
	case sideLeft:
		if s.right[rel] {
			return true
		}
		s.left[rel] = true
		return false
	case sideRight:
		if s.left[rel] {
			return true
		}
		s.right[rel] = true
		return false
	default:
		return false
	}
}
