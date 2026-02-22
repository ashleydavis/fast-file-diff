package lib

import (
	"path/filepath"
	"sync"
)

// Side indicates which tree (left or right) a path was seen on.
type Side int

const (
	SideLeft Side = iota
	SideRight
)

// PairJob is a single pair to compare (relative path); compare phase stats both files.
type PairJob struct {
	Rel string
}

// DiscoveredSet tracks which relative paths have been seen on left and right.
// leftOnlyCount and rightOnlyCount are maintained in Add() so counts are O(1).
type DiscoveredSet struct {
	mu             sync.Mutex
	pool           *PathPool
	left           map[string]bool
	right          map[string]bool
	pairPaths      []string
	leftOnlyCount  int
	rightOnlyCount int
}

// NewDiscoveredSet returns a new discovered set using the given path pool.
func NewDiscoveredSet(pool *PathPool) *DiscoveredSet {
	return &DiscoveredSet{
		pool:  pool,
		left:  make(map[string]bool),
		right: make(map[string]bool),
	}
}

// Add records that rel was seen on the given side. Returns true when this completes a pair (the other side had already been seen for rel).
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
				discoveredSet.rightOnlyCount-- // was right-only, now a pair
			}
			return firstTime
		}
		discoveredSet.left[rel] = true
		discoveredSet.leftOnlyCount++
		return false
	case SideRight:
		if discoveredSet.left[rel] {
			firstTime := !discoveredSet.right[rel]
			discoveredSet.right[rel] = true
			if firstTime {
				discoveredSet.pairPaths = append(discoveredSet.pairPaths, rel)
				discoveredSet.leftOnlyCount-- // was left-only, now a pair
			}
			return firstTime
		}
		discoveredSet.right[rel] = true
		discoveredSet.rightOnlyCount++
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

// LeftOnlyCount returns the number of paths seen on left but not on right. O(1).
func (discoveredSet *DiscoveredSet) LeftOnlyCount() int {
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	return discoveredSet.leftOnlyCount
}

// RightOnlyCount returns the number of paths seen on right but not on left. O(1).
func (discoveredSet *DiscoveredSet) RightOnlyCount() int {
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	return discoveredSet.rightOnlyCount
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
