package lib

import (
	"path/filepath"
	"sync"
	"time"
)

// Side indicates which tree (left or right) a path was seen on.
type Side int

const (
	SideLeft  Side = iota
	SideRight
)

// PairInfo holds cached size and mtime for both sides of a pair (from the discovery walk).
type PairInfo struct {
	LeftSize   int64
	LeftMtime  time.Time
	RightSize  int64
	RightMtime time.Time
}

// PairJob is a single pair to compare; Cached may be nil (compare will stat).
type PairJob struct {
	Rel    string
	Cached *PairInfo
}

type fileInfoCache struct {
	size int64
	mtime time.Time
}

// DiscoveredSet tracks which relative paths have been seen on left and right,
// and caches size and mtime from the walk for use during compare.
type DiscoveredSet struct {
	mu           sync.Mutex
	pool         *PathPool
	left         map[string]bool
	right        map[string]bool
	leftFileInfo map[string]fileInfoCache
	rightFileInfo map[string]fileInfoCache
	pairPaths    []string
}

// NewDiscoveredSet returns a new discovered set using the given path pool.
func NewDiscoveredSet(pool *PathPool) *DiscoveredSet {
	return &DiscoveredSet{
		pool:          pool,
		left:          make(map[string]bool),
		right:         make(map[string]bool),
		leftFileInfo:  make(map[string]fileInfoCache),
		rightFileInfo: make(map[string]fileInfoCache),
	}
}

// Add records that rel was seen on the given side with the given size and mtime (from the walk).
// It returns true when this completes a pair (the other side had already been seen for rel).
func (discoveredSet *DiscoveredSet) Add(rel string, side Side, size int64, mtime time.Time) bool {
	rel = discoveredSet.pool.Intern(filepath.Clean(filepath.ToSlash(rel)))
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	info := fileInfoCache{size: size, mtime: mtime.Truncate(time.Second)}
	switch side {
	case SideLeft:
		discoveredSet.leftFileInfo[rel] = info
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
		discoveredSet.rightFileInfo[rel] = info
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

// PairCachedInfo returns cached size and mtime for both sides of the pair, if present.
func (discoveredSet *DiscoveredSet) PairCachedInfo(rel string) (*PairInfo, bool) {
	discoveredSet.mu.Lock()
	defer discoveredSet.mu.Unlock()
	leftInfo, hasLeft := discoveredSet.leftFileInfo[rel]
	rightInfo, hasRight := discoveredSet.rightFileInfo[rel]
	if !hasLeft || !hasRight {
		return nil, false
	}
	return &PairInfo{
		LeftSize:   leftInfo.size,
		LeftMtime:  leftInfo.mtime,
		RightSize:  rightInfo.size,
		RightMtime: rightInfo.mtime,
	}, true
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
