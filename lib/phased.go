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

// ValidPhaseNames is the ordered list of phase names for the phased pipeline (walk-left through compare-hashes).
var ValidPhaseNames = []string{"walk-left", "walk-right", "build-pairs", "classify-pairs", "hash-left", "hash-right", "compare-hashes"}

// ValidPhase returns true if name is one of the valid phase names.
func ValidPhase(name string) bool {
	for _, n := range ValidPhaseNames {
		if n == name {
			return true
		}
	}
	return false
}

// PhaseWalkLeft walks the left tree and returns FileInfo for every file. Stub returns nil until implemented.
func PhaseWalkLeft(leftRoot string, dirBatchSize int) []FileInfo {
	return nil
}

// PhaseWalkRight walks the right tree and returns FileInfo for every file. Stub returns nil until implemented.
func PhaseWalkRight(rightRoot string, dirBatchSize int) []FileInfo {
	return nil
}

// PhaseBuildPairs builds pairs and left-only/right-only path lists from left and right FileInfo slices. Stub returns zero value until implemented.
func PhaseBuildPairs(left, right []FileInfo) BuildPairsResult {
	return BuildPairsResult{}
}

// PhaseClassifyPairs classifies pairs by size/mtime into differing-by-size, content-check queue, and same. Stub returns zero value until implemented.
func PhaseClassifyPairs(pairs []*Pair) ClassifyPairsResult {
	return ClassifyPairsResult{}
}

// PhaseHashLeft hashes the left-side file for each pair in contentCheckQueue and sets Pair.Left.Hash. Stub is a no-op until implemented.
func PhaseHashLeft(leftRoot string, contentCheckQueue []*Pair, hashAlg string, threshold int) {
}

// PhaseHashRight hashes the right-side file for each pair in contentCheckQueue and sets Pair.Right.Hash. Stub is a no-op until implemented.
func PhaseHashRight(rightRoot string, contentCheckQueue []*Pair, hashAlg string, threshold int) {
}

// PhaseCompareHashes produces DiffResults from classified pairs and left-only/right-only paths. Stub returns nil until implemented.
func PhaseCompareHashes(contentCheckQueue, differingBySize []*Pair, leftOnlyPaths, rightOnlyPaths []string, leftByPath, rightByPath map[string]*FileInfo) []DiffResult {
	return nil
}
