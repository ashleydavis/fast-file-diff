package lib

import (
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
