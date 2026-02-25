package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRequireZeroOrTwoArgs(t *testing.T) {
	cmd := &cobra.Command{}
	if err := requireZeroOrTwoArgs(cmd, nil); err != nil {
		t.Errorf("requireZeroOrTwoArgs(nil) = %v", err)
	}
	if err := requireZeroOrTwoArgs(cmd, []string{"a", "b"}); err != nil {
		t.Errorf("requireZeroOrTwoArgs([a,b]) = %v", err)
	}
	if err := requireZeroOrTwoArgs(cmd, []string{"only"}); err == nil {
		t.Error("requireZeroOrTwoArgs([one]) want error")
	}
	if err := requireZeroOrTwoArgs(cmd, []string{"a", "b", "c"}); err == nil {
		t.Error("requireZeroOrTwoArgs([a,b,c]) want error")
	}
}

func TestEstimateRemainingFromElapsed(t *testing.T) {
	elapsed := 10 * time.Second
	// 10 processed in 10s => 1s per pair; 5 pending => 5s remaining
	got := estimateRemainingFromElapsed(elapsed, 10, 5)
	if got != 5*time.Second {
		t.Errorf("estimateRemainingFromElapsed(10s, 10, 5) = %v, want 5s", got)
	}
	// processed 0 => no estimate
	got = estimateRemainingFromElapsed(elapsed, 0, 5)
	if got != 0 {
		t.Errorf("estimateRemainingFromElapsed(10s, 0, 5) = %v, want 0", got)
	}
	// pending 0 => 0 remaining
	got = estimateRemainingFromElapsed(elapsed, 10, 0)
	if got != 0 {
		t.Errorf("estimateRemainingFromElapsed(10s, 10, 0) = %v, want 0", got)
	}
}

// runLsCapture runs "ffd ls dir" and returns the list of lines printed to stdout (sorted).
func runLsCapture(t *testing.T, dir string) []string {
	t.Helper()
	var out bytes.Buffer
	rootCmd.SetArgs([]string{"ls", dir})
	rootCmd.SetOut(&out)
	rootCmd.SetErr(os.Stderr)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("ffd ls %q: %v", dir, err)
	}
	s := strings.TrimSpace(out.String())
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	sort.Strings(lines)
	return lines
}

func TestLs_emptyDirectory(t *testing.T) {
	dir := t.TempDir()
	got := runLsCapture(t, dir)
	if len(got) != 0 {
		t.Errorf("ls empty dir: got %q, want no lines", got)
	}
}

func TestLs_oneFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	got := runLsCapture(t, dir)
	want := []string{"a"}
	if len(got) != len(want) || (len(got) > 0 && got[0] != want[0]) {
		t.Errorf("ls one file: got %q, want %q", got, want)
	}
}

func TestLs_twoFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	got := runLsCapture(t, dir)
	want := []string{"a", "b"}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Errorf("ls two files: got %q, want %q", got, want)
	} else {
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("ls two files: got %q, want %q", got, want)
				break
			}
		}
	}
}

func TestLs_directoryWithEmptySubdirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	got := runLsCapture(t, dir)
	if len(got) != 0 {
		t.Errorf("ls dir with empty subdir: got %q, want no files (we only list files)", got)
	}
}

func TestLs_directoryWithFileAndSubdirectoryWithFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "top"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	got := runLsCapture(t, dir)
	want := []string{filepath.Join("sub", "nested"), "top"}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Errorf("ls file+subdir with file: got %q, want %q", got, want)
	} else {
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("ls file+subdir with file: got %q, want %q", got, want)
				break
			}
		}
	}
}

// TestOnlySideLogLines_emptyPaths returns nil when paths is empty.
func TestOnlySideLogLines_emptyPaths(t *testing.T) {
	dir := t.TempDir()
	got := onlySideLogLines(dir, nil, "left")
	if got != nil {
		t.Errorf("onlySideLogLines(_, nil, _) = %v, want nil", got)
	}
	got = onlySideLogLines(dir, []string{}, "right")
	if got != nil {
		t.Errorf("onlySideLogLines(_, [], _) = %v, want nil", got)
	}
}

// TestOnlySideLogLines_singleFile returns one indented line for a single file path.
func TestOnlySideLogLines_singleFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	got := onlySideLogLines(dir, []string{"f"}, "left")
	want := []string{"  f"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("onlySideLogLines(single file) = %v, want %v", got, want)
	}
}

// TestOnlySideLogLines_directoryWithOneFile collapses to "directory X is left only and contains 1 file(s)".
func TestOnlySideLogLines_directoryWithOneFile(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	paths := []string{"sub", "sub/a"}
	got := onlySideLogLines(dir, paths, "left")
	want := []string{"directory sub is left only and contains 1 file(s)"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("onlySideLogLines(dir with one file) = %v, want %v", got, want)
	}
}

// TestOnlySideLogLines_directoryWithTwoFiles collapses to one line with "contains 2 file(s)".
func TestOnlySideLogLines_directoryWithTwoFiles(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(sub, name), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	paths := []string{"sub", "sub/a", "sub/b"}
	got := onlySideLogLines(dir, paths, "right")
	want := []string{"directory sub is right only and contains 2 file(s)"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("onlySideLogLines(dir with two files) = %v, want %v", got, want)
	}
}

// TestOnlySideLogLines_fileAndDirectory returns file line plus collapsed directory line.
func TestOnlySideLogLines_fileAndDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "alone"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "n"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	paths := []string{"alone", "sub", "sub/n"}
	got := onlySideLogLines(dir, paths, "left")
	if len(got) != 2 {
		t.Fatalf("onlySideLogLines(file+dir) len = %d, want 2; got %v", len(got), got)
	}
	if got[0] != "  alone" {
		t.Errorf("first line = %q, want %q", got[0], "  alone")
	}
	if got[1] != "directory sub is left only and contains 1 file(s)" {
		t.Errorf("second line = %q, want directory summary", got[1])
	}
}
