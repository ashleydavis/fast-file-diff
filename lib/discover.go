package lib

import (
	"path/filepath"
	"sync"
)

// Side indicates which tree (left or right) a path was seen on.
type Side int

const (
	SideLeft  Side = iota
	SideRight
)

// DiscoveredSet tracks which relative paths have been seen on left and right.
// When the same path is seen on both sides, Add returns true (form a pair).
type DiscoveredSet struct {
	mu    sync.Mutex
	pool  *PathPool
	left  map[string]bool
	right map[string]bool
}

// NewDiscoveredSet returns a new discovered set using the given path pool.
func NewDiscoveredSet(pool *PathPool) *DiscoveredSet {
	return &DiscoveredSet{
		pool:  pool,
		left:  make(map[string]bool),
		right: make(map[string]bool),
	}
}

// Add records that rel was seen on the given side. It returns true when this
// completes a pair (the other side had already been seen for rel).
func (s *DiscoveredSet) Add(rel string, sd Side) bool {
	rel = s.pool.Intern(filepath.Clean(filepath.ToSlash(rel)))
	s.mu.Lock()
	defer s.mu.Unlock()
	switch sd {
	case SideLeft:
		if s.right[rel] {
			firstTime := !s.left[rel]
			s.left[rel] = true
			return firstTime
		}
		s.left[rel] = true
		return false
	case SideRight:
		if s.left[rel] {
			firstTime := !s.right[rel]
			s.right[rel] = true
			return firstTime
		}
		s.right[rel] = true
		return false
	default:
		return false
	}
}

// LeftOnlyPaths returns relative paths that were seen on left but not on right.
func (s *DiscoveredSet) LeftOnlyPaths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for rel := range s.left {
		if !s.right[rel] {
			out = append(out, rel)
		}
	}
	return out
}

// RightOnlyPaths returns relative paths that were seen on right but not on left.
func (s *DiscoveredSet) RightOnlyPaths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for rel := range s.right {
		if !s.left[rel] {
			out = append(out, rel)
		}
	}
	return out
}
