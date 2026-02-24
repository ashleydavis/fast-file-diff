package lib

import "time"

// FileInfo holds metadata for one discovered file: relative path, size, mtime (normalized to second), and optional hash.
// Used by the phased pipeline; Hash is filled in only for files that need content comparison (hash-left, hash-right).
type FileInfo struct {
	Rel   string    // relative path
	Size  int64     // file size in bytes
	Mtime time.Time // modification time, normalized to 1-second granularity
	Hash  string    // hex hash of file content; empty until hash-left or hash-right sets it
}

// NormalizeMtime truncates t to 1-second granularity for consistent comparison across filesystems.
func NormalizeMtime(t time.Time) time.Time {
	return t.Truncate(time.Second)
}

// Pair represents one path present on both sides; Left and Right point into the left/right FileInfo slices.
type Pair struct {
	Rel   string    // relative path
	Left  *FileInfo // file info from left tree (same path)
	Right *FileInfo // file info from right tree (same path)
}

// BuildPairsResult is the output of the build-pairs phase: paths only on left, only on right, and pairs (both sides).
type BuildPairsResult struct {
	LeftOnlyPaths  []string // relative paths that appear only in the left array
	RightOnlyPaths []string // relative paths that appear only in the right array
	Pairs          []*Pair  // paths that appear on both sides; each pair references left and right FileInfo
}

// ClassifyPairsResult is the output of the classify-pairs phase: pairs differing by size, pairs needing content check, and pairs same by size+mtime.
type ClassifyPairsResult struct {
	DifferingBySize   []*Pair // pairs where left and right size differ (no hash needed)
	ContentCheckQueue []*Pair // pairs where size is equal but mtime differs (need hash)
	SameBySizeMtime   []*Pair // pairs where size and mtime are equal (skip content check)
}
