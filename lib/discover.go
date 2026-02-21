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
// When the same path is seen on both sides, Add returns true (form a pair)
// and the path is appended to the list returned by PairPaths().
type DiscoveredSet struct {
	mu        sync.Mutex
	pool      *PathPool
	left      map[string]bool
	right     map[string]bool
	pairPaths []string // paths that have been seen on both sides, in discovery order
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
func (discoveredSet *DiscoveredSet) Add(rel string, side Side) bool {
	rel = discoveredSet.pool.Intern(filepath.Clean(filepath.ToSlash(rel)))
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	switch side {
	case SideLeft:
		if discoveredSet.right[rel] {
			firstTime := !discoveredSet.left[rel]
			discoveredSet.left[rel] = true
			if firstTime {
				discoveredSet.pairPaths = append(discoveredSet.pairPaths, rel)
			}
			return firstTime
		}
		discoveredSet.left[rel] = true
		return false
	case SideRight:
		if discoveredSet.left[rel] {
			firstTime := !discoveredSet.right[rel]
			discoveredSet.right[rel] = true
			if firstTime {
				discoveredSet.pairPaths = append(discoveredSet.pairPaths, rel)
			}
			return firstTime
		}
		discoveredSet.right[rel] = true
		return false
	default:
		return false
	}
}

// PairsCount returns the number of file pairs discovered so far (both sides seen).
func (discoveredSet *DiscoveredSet) PairsCount() int {
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	return len(discoveredSet.pairPaths)
}

// PairPaths returns a copy of the relative paths that form pairs (seen on both sides), in discovery order.
func (discoveredSet *DiscoveredSet) PairPaths() []string {
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	out := make([]string, len(discoveredSet.pairPaths))
	copy(out, discoveredSet.pairPaths)
	return out
}

// LeftOnlyPaths returns relative paths that were seen on left but not on right.
func (discoveredSet *DiscoveredSet) LeftOnlyPaths() []string {
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	var out []string
	for rel := range discoveredSet.left {
		if !discoveredSet.right[rel] {
			out = append(out, rel)
		}
	}
	return out
}

// RightOnlyPaths returns relative paths that were seen on right but not on left.
func (discoveredSet *DiscoveredSet) RightOnlyPaths() []string {
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	var out []string
	for rel := range discoveredSet.right {
		if !discoveredSet.left[rel] {
			out = append(out, rel)
		}
	}
	return out
}
