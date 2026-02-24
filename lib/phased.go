package lib

import (
	"path/filepath"
	"time"
)

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

// PhaseWalkLeft walks the left tree and returns FileInfo for every file.
func PhaseWalkLeft(leftRoot string, dirBatchSize int) []FileInfo {
	return WalkTreeCollectFileInfo(leftRoot, dirBatchSize)
}

// PhaseWalkRight walks the right tree and returns FileInfo for every file.
func PhaseWalkRight(rightRoot string, dirBatchSize int) []FileInfo {
	return WalkTreeCollectFileInfo(rightRoot, dirBatchSize)
}

// PhaseBuildPairs builds pairs and left-only/right-only path lists from left and right FileInfo slices.
func PhaseBuildPairs(left, right []FileInfo) BuildPairsResult {
	leftByPath := make(map[string]*FileInfo)
	for i := range left {
		leftByPath[left[i].Rel] = &left[i]
	}
	rightByPath := make(map[string]*FileInfo)
	for i := range right {
		rightByPath[right[i].Rel] = &right[i]
	}
	var leftOnlyPaths []string
	for i := range left {
		rel := left[i].Rel
		if rightByPath[rel] == nil {
			leftOnlyPaths = append(leftOnlyPaths, rel)
		}
	}
	var rightOnlyPaths []string
	for i := range right {
		rel := right[i].Rel
		if leftByPath[rel] == nil {
			rightOnlyPaths = append(rightOnlyPaths, rel)
		}
	}
	var pairs []*Pair
	for rel, leftInfo := range leftByPath {
		if rightInfo := rightByPath[rel]; rightInfo != nil {
			pairs = append(pairs, &Pair{Rel: rel, Left: leftInfo, Right: rightInfo})
		}
	}
	return BuildPairsResult{
		LeftOnlyPaths:  leftOnlyPaths,
		RightOnlyPaths: rightOnlyPaths,
		Pairs:          pairs,
	}
}

// PhaseClassifyPairs classifies pairs by size/mtime into differing-by-size, content-check queue, and same.
func PhaseClassifyPairs(pairs []*Pair) ClassifyPairsResult {
	var differingBySize []*Pair
	var contentCheckQueue []*Pair
	var sameBySizeMtime []*Pair
	for _, pair := range pairs {
		if pair.Left.Size != pair.Right.Size {
			differingBySize = append(differingBySize, pair)
			continue
		}
		if pair.Left.Mtime.Equal(pair.Right.Mtime) {
			sameBySizeMtime = append(sameBySizeMtime, pair)
			continue
		}
		contentCheckQueue = append(contentCheckQueue, pair)
	}
	return ClassifyPairsResult{
		DifferingBySize:   differingBySize,
		ContentCheckQueue: contentCheckQueue,
		SameBySizeMtime:   sameBySizeMtime,
	}
}

// hashContentCheckQueue loads and hashes the file at root+rel for each pair and assigns the hash to the left or right FileInfo. Used by PhaseHashLeft and PhaseHashRight.
func hashContentCheckQueue(root string, contentCheckQueue []*Pair, hashAlg string, threshold int, assignToLeft bool) {
	for _, pair := range contentCheckQueue {
		path := filepath.Join(root, pair.Rel)
		hash, err := HashFile(path, hashAlg, threshold)
		if err != nil {
			continue // non-fatal: leave Hash empty; compare-hashes can report as error
		}
		if assignToLeft {
			pair.Left.Hash = hash
		} else {
			pair.Right.Hash = hash
		}
	}
}

// PhaseHashLeft hashes the left-side file for each pair in contentCheckQueue and sets Pair.Left.Hash.
func PhaseHashLeft(leftRoot string, contentCheckQueue []*Pair, hashAlg string, threshold int) {
	hashContentCheckQueue(leftRoot, contentCheckQueue, hashAlg, threshold, true)
}

// PhaseHashRight hashes the right-side file for each pair in contentCheckQueue and sets Pair.Right.Hash.
func PhaseHashRight(rightRoot string, contentCheckQueue []*Pair, hashAlg string, threshold int) {
	hashContentCheckQueue(rightRoot, contentCheckQueue, hashAlg, threshold, false)
}

// PhaseCompareHashes produces DiffResults from classified pairs and left-only/right-only paths.
func PhaseCompareHashes(contentCheckQueue, differingBySize []*Pair, leftOnlyPaths, rightOnlyPaths []string, leftByPath, rightByPath map[string]*FileInfo) []DiffResult {
	var diffs []DiffResult
	for _, pair := range differingBySize {
		diffs = append(diffs, DiffResult{
			Rel:        pair.Rel,
			Reason:     "size changed",
			LeftHash:   pair.Left.Hash,
			RightHash:  pair.Right.Hash,
			LeftSize:   pair.Left.Size,
			RightSize:  pair.Right.Size,
			LeftMtime:  pair.Left.Mtime,
			RightMtime: pair.Right.Mtime,
		})
	}
	for _, pair := range contentCheckQueue {
		if pair.Left.Hash != pair.Right.Hash {
			diffs = append(diffs, DiffResult{
				Rel:        pair.Rel,
				Reason:     "content differs",
				LeftHash:   pair.Left.Hash,
				RightHash:  pair.Right.Hash,
				LeftSize:   pair.Left.Size,
				RightSize:  pair.Right.Size,
				LeftMtime:  pair.Left.Mtime,
				RightMtime: pair.Right.Mtime,
			})
		}
	}
	for _, rel := range leftOnlyPaths {
		info := leftByPath[rel]
		if info != nil {
			diffs = append(diffs, DiffResult{
				Rel:      rel,
				Reason:   "left only",
				LeftSize: info.Size, LeftMtime: info.Mtime,
				LeftOnly: true,
			})
		}
	}
	for _, rel := range rightOnlyPaths {
		info := rightByPath[rel]
		if info != nil {
			diffs = append(diffs, DiffResult{
				Rel:       rel,
				Reason:    "right only",
				RightSize: info.Size, RightMtime: info.Mtime,
			})
		}
	}
	return diffs
}
