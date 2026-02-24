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
