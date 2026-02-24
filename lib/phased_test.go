package lib

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeMtime_truncatesToSecond(t *testing.T) {
	// Nanoseconds should be truncated to zero.
	withNanos := time.Unix(1000, 123456789)
	got := NormalizeMtime(withNanos)
	want := time.Unix(1000, 0)
	if !got.Equal(want) {
		t.Errorf("NormalizeMtime(%v) = %v, want %v", withNanos, got, want)
	}
}

func TestNormalizeMtime_preservesSecond(t *testing.T) {
	alreadySecond := time.Unix(2000, 0)
	got := NormalizeMtime(alreadySecond)
	if !got.Equal(alreadySecond) {
		t.Errorf("NormalizeMtime(%v) = %v, want unchanged", alreadySecond, got)
	}
}

func TestFileInfo_constructAndUseInSlice(t *testing.T) {
	// FileInfo is a struct; ensure we can construct it and use it in a slice (per plan).
	mtime := NormalizeMtime(time.Unix(1, 0))
	slice := []FileInfo{
		{Rel: "a", Size: 10, Mtime: mtime, Hash: ""},
		{Rel: "b", Size: 20, Mtime: mtime, Hash: "abc"},
	}
	if len(slice) != 2 {
		t.Fatalf("len(slice) = %d, want 2", len(slice))
	}
	if slice[0].Rel != "a" || slice[0].Size != 10 || slice[0].Hash != "" {
		t.Errorf("slice[0] = %+v", slice[0])
	}
	if slice[1].Rel != "b" || slice[1].Hash != "abc" {
		t.Errorf("slice[1] = %+v", slice[1])
	}
}

func TestBuildPairsResult_zeroValue(t *testing.T) {
	var result BuildPairsResult
	if result.LeftOnlyPaths != nil || result.RightOnlyPaths != nil || result.Pairs != nil {
		t.Errorf("zero value should have nil slices; got %+v", result)
	}
}

func TestClassifyPairsResult_zeroValue(t *testing.T) {
	var result ClassifyPairsResult
	if result.DifferingBySize != nil || result.ContentCheckQueue != nil || result.SameBySizeMtime != nil {
		t.Errorf("zero value should have nil slices; got %+v", result)
	}
}

func TestValidPhase(t *testing.T) {
	for _, name := range ValidPhaseNames {
		if !ValidPhase(name) {
			t.Errorf("ValidPhase(%q) = false, want true", name)
		}
	}
	invalid := []string{"", "walk", "1", "walk_left", "compare"}
	for _, name := range invalid {
		if ValidPhase(name) {
			t.Errorf("ValidPhase(%q) = true, want false", name)
		}
	}
}

func TestWalkTreeCollectFileInfo_emptyDir(t *testing.T) {
	dir := t.TempDir()
	got := WalkTreeCollectFileInfo(dir, 4096)
	if len(got) != 0 {
		t.Errorf("empty dir: len = %d, want 0", len(got))
	}
}

func TestWalkTreeCollectFileInfo_oneFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	got := WalkTreeCollectFileInfo(dir, 4096)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Rel != "a.txt" {
		t.Errorf("Rel = %q, want a.txt", got[0].Rel)
	}
	if got[0].Size != 5 {
		t.Errorf("Size = %d, want 5", got[0].Size)
	}
	if got[0].Hash != "" {
		t.Errorf("Hash = %q, want empty", got[0].Hash)
	}
	// Mtime normalized to second
	if got[0].Mtime.UnixNano()%int64(time.Second) != 0 {
		t.Errorf("Mtime should be normalized to second, got %v", got[0].Mtime)
	}
}

func TestWalkTreeCollectFileInfo_subdir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "top.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "nested.txt"), []byte("y"), 0644); err != nil {
		t.Fatal(err)
	}
	got := WalkTreeCollectFileInfo(dir, 4096)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	paths := make(map[string]int64)
	for _, f := range got {
		paths[f.Rel] = f.Size
	}
	if paths["top.txt"] != 1 || paths[filepath.Join("sub", "nested.txt")] != 1 {
		t.Errorf("unexpected paths or sizes: %v", paths)
	}
}

func TestPhaseWalkLeft_and_PhaseWalkRight_useSameBehavior(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	left := PhaseWalkLeft(dir, 4096)
	right := PhaseWalkRight(dir, 4096)
	if len(left) != 1 || len(right) != 1 {
		t.Fatalf("PhaseWalkLeft len=%d, PhaseWalkRight len=%d, want 1 each", len(left), len(right))
	}
	if left[0].Rel != right[0].Rel || left[0].Size != right[0].Size {
		t.Errorf("same root should yield same Rel/Size: left %+v, right %+v", left[0], right[0])
	}
}

func TestPhaseBuildPairs_leftOnlyRightOnlyPairs(t *testing.T) {
	mtime := NormalizeMtime(time.Unix(1, 0))
	left := []FileInfo{
		{Rel: "a", Size: 1, Mtime: mtime, Hash: ""},
		{Rel: "b", Size: 2, Mtime: mtime, Hash: ""},
	}
	right := []FileInfo{
		{Rel: "b", Size: 2, Mtime: mtime, Hash: ""},
		{Rel: "c", Size: 3, Mtime: mtime, Hash: ""},
	}
	got := PhaseBuildPairs(left, right)
	if len(got.LeftOnlyPaths) != 1 || got.LeftOnlyPaths[0] != "a" {
		t.Errorf("LeftOnlyPaths = %v, want [a]", got.LeftOnlyPaths)
	}
	if len(got.RightOnlyPaths) != 1 || got.RightOnlyPaths[0] != "c" {
		t.Errorf("RightOnlyPaths = %v, want [c]", got.RightOnlyPaths)
	}
	if len(got.Pairs) != 1 {
		t.Fatalf("Pairs len = %d, want 1", len(got.Pairs))
	}
	p := got.Pairs[0]
	if p.Rel != "b" || p.Left != &left[1] || p.Right != &right[0] {
		t.Errorf("pair: Rel=%q Left=%p Right=%p; want Rel=b Left=%p Right=%p", p.Rel, p.Left, p.Right, &left[1], &right[0])
	}
}

func TestPhaseBuildPairs_bothEmpty(t *testing.T) {
	got := PhaseBuildPairs(nil, nil)
	if got.LeftOnlyPaths != nil || got.RightOnlyPaths != nil || got.Pairs != nil {
		t.Errorf("both empty should have nil slices: %+v", got)
	}
}

func TestPhaseBuildPairs_oneSideEmpty(t *testing.T) {
	left := []FileInfo{{Rel: "x", Size: 0, Mtime: time.Time{}, Hash: ""}}
	got := PhaseBuildPairs(left, nil)
	if len(got.LeftOnlyPaths) != 1 || got.LeftOnlyPaths[0] != "x" {
		t.Errorf("LeftOnlyPaths = %v", got.LeftOnlyPaths)
	}
	if len(got.RightOnlyPaths) != 0 {
		t.Errorf("RightOnlyPaths = %v, want empty", got.RightOnlyPaths)
	}
	if len(got.Pairs) != 0 {
		t.Errorf("Pairs = %v, want empty", got.Pairs)
	}
}

func TestPhaseClassifyPairs_sameSizeSameMtime_toSameBySizeMtime(t *testing.T) {
	mtime := NormalizeMtime(time.Unix(1, 0))
	left := FileInfo{Rel: "a", Size: 10, Mtime: mtime, Hash: ""}
	right := FileInfo{Rel: "a", Size: 10, Mtime: mtime, Hash: ""}
	pairs := []*Pair{{Rel: "a", Left: &left, Right: &right}}
	got := PhaseClassifyPairs(pairs)
	if len(got.SameBySizeMtime) != 1 || len(got.DifferingBySize) != 0 || len(got.ContentCheckQueue) != 0 {
		t.Errorf("DifferingBySize=%d ContentCheckQueue=%d SameBySizeMtime=%d, want 0,0,1", len(got.DifferingBySize), len(got.ContentCheckQueue), len(got.SameBySizeMtime))
	}
}

func TestPhaseClassifyPairs_differentSize_toDifferingBySize(t *testing.T) {
	mtime := NormalizeMtime(time.Unix(1, 0))
	left := FileInfo{Rel: "a", Size: 10, Mtime: mtime, Hash: ""}
	right := FileInfo{Rel: "a", Size: 20, Mtime: mtime, Hash: ""}
	pairs := []*Pair{{Rel: "a", Left: &left, Right: &right}}
	got := PhaseClassifyPairs(pairs)
	if len(got.DifferingBySize) != 1 || len(got.ContentCheckQueue) != 0 || len(got.SameBySizeMtime) != 0 {
		t.Errorf("DifferingBySize=%d ContentCheckQueue=%d SameBySizeMtime=%d, want 1,0,0", len(got.DifferingBySize), len(got.ContentCheckQueue), len(got.SameBySizeMtime))
	}
}

func TestPhaseClassifyPairs_sameSizeDifferentMtime_toContentCheckQueue(t *testing.T) {
	left := FileInfo{Rel: "a", Size: 10, Mtime: NormalizeMtime(time.Unix(1, 0)), Hash: ""}
	right := FileInfo{Rel: "a", Size: 10, Mtime: NormalizeMtime(time.Unix(2, 0)), Hash: ""}
	pairs := []*Pair{{Rel: "a", Left: &left, Right: &right}}
	got := PhaseClassifyPairs(pairs)
	if len(got.ContentCheckQueue) != 1 || len(got.DifferingBySize) != 0 || len(got.SameBySizeMtime) != 0 {
		t.Errorf("DifferingBySize=%d ContentCheckQueue=%d SameBySizeMtime=%d, want 0,1,0", len(got.DifferingBySize), len(got.ContentCheckQueue), len(got.SameBySizeMtime))
	}
}

func TestPhaseClassifyPairs_mix(t *testing.T) {
	m1 := NormalizeMtime(time.Unix(1, 0))
	m2 := NormalizeMtime(time.Unix(2, 0))
	diffSizeL := FileInfo{Rel: "diff", Size: 1, Mtime: m1, Hash: ""}
	diffSizeR := FileInfo{Rel: "diff", Size: 2, Mtime: m1, Hash: ""}
	sameL := FileInfo{Rel: "same", Size: 5, Mtime: m1, Hash: ""}
	sameR := FileInfo{Rel: "same", Size: 5, Mtime: m1, Hash: ""}
	contentL := FileInfo{Rel: "content", Size: 5, Mtime: m1, Hash: ""}
	contentR := FileInfo{Rel: "content", Size: 5, Mtime: m2, Hash: ""}
	pairs := []*Pair{
		{Rel: "diff", Left: &diffSizeL, Right: &diffSizeR},
		{Rel: "same", Left: &sameL, Right: &sameR},
		{Rel: "content", Left: &contentL, Right: &contentR},
	}
	got := PhaseClassifyPairs(pairs)
	if len(got.DifferingBySize) != 1 || len(got.SameBySizeMtime) != 1 || len(got.ContentCheckQueue) != 1 {
		t.Errorf("DifferingBySize=%d SameBySizeMtime=%d ContentCheckQueue=%d, want 1,1,1", len(got.DifferingBySize), len(got.SameBySizeMtime), len(got.ContentCheckQueue))
	}
}

func TestPhaseHashLeft_and_PhaseHashRight_setHashes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	left := FileInfo{Rel: "f", Size: 5, Mtime: time.Time{}, Hash: ""}
	right := FileInfo{Rel: "f", Size: 5, Mtime: time.Time{}, Hash: ""}
	pairs := []*Pair{{Rel: "f", Left: &left, Right: &right}}
	PhaseHashLeft(dir, pairs, "xxhash", 10*1024*1024)
	PhaseHashRight(dir, pairs, "xxhash", 10*1024*1024)
	if left.Hash == "" || right.Hash == "" {
		t.Errorf("hashes not set: Left.Hash=%q Right.Hash=%q", left.Hash, right.Hash)
	}
	if left.Hash != right.Hash {
		t.Errorf("same file should have same hash: %q vs %q", left.Hash, right.Hash)
	}
}

func TestPhaseCompareHashes_contentCheckSameHash_notInResult(t *testing.T) {
	mtime := NormalizeMtime(time.Unix(1, 0))
	left := FileInfo{Rel: "a", Size: 5, Mtime: mtime, Hash: "abc"}
	right := FileInfo{Rel: "a", Size: 5, Mtime: mtime, Hash: "abc"}
	contentCheck := []*Pair{{Rel: "a", Left: &left, Right: &right}}
	leftByPath := map[string]*FileInfo{"a": &left}
	rightByPath := map[string]*FileInfo{"a": &right}
	got := PhaseCompareHashes(contentCheck, nil, nil, nil, leftByPath, rightByPath)
	for _, d := range got {
		if d.Rel == "a" {
			t.Errorf("same hash should not produce diff for a, got Reason=%q", d.Reason)
		}
	}
}

func TestPhaseCompareHashes_contentCheckDifferentHash_inResult(t *testing.T) {
	mtime := NormalizeMtime(time.Unix(1, 0))
	left := FileInfo{Rel: "a", Size: 5, Mtime: mtime, Hash: "left"}
	right := FileInfo{Rel: "a", Size: 5, Mtime: mtime, Hash: "right"}
	contentCheck := []*Pair{{Rel: "a", Left: &left, Right: &right}}
	leftByPath := map[string]*FileInfo{"a": &left}
	rightByPath := map[string]*FileInfo{"a": &right}
	got := PhaseCompareHashes(contentCheck, nil, nil, nil, leftByPath, rightByPath)
	if len(got) != 1 || got[0].Rel != "a" || got[0].Reason != "content differs" {
		t.Errorf("got %v, want single DiffResult Rel=a Reason=content differs", got)
	}
}

func TestPhaseCompareHashes_differingBySize_and_leftOnly_rightOnly_inResult(t *testing.T) {
	mtime := NormalizeMtime(time.Unix(1, 0))
	leftA := FileInfo{Rel: "size", Size: 1, Mtime: mtime, Hash: ""}
	rightA := FileInfo{Rel: "size", Size: 2, Mtime: mtime, Hash: ""}
	leftOnly := FileInfo{Rel: "lonly", Size: 10, Mtime: mtime, Hash: ""}
	rightOnly := FileInfo{Rel: "ronly", Size: 20, Mtime: mtime, Hash: ""}
	differingBySize := []*Pair{{Rel: "size", Left: &leftA, Right: &rightA}}
	leftByPath := map[string]*FileInfo{"size": &leftA, "lonly": &leftOnly}
	rightByPath := map[string]*FileInfo{"size": &rightA, "ronly": &rightOnly}
	got := PhaseCompareHashes(nil, differingBySize, []string{"lonly"}, []string{"ronly"}, leftByPath, rightByPath)
	if len(got) != 3 {
		t.Fatalf("want 3 diffs (size, left-only, right-only), got %d", len(got))
	}
	reasons := make(map[string]string)
	for _, d := range got {
		reasons[d.Rel] = d.Reason
	}
	if reasons["size"] != "size changed" || reasons["lonly"] != "left only" || reasons["ronly"] != "right only" {
		t.Errorf("reasons = %v", reasons)
	}
}
