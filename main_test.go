package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

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
