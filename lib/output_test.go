package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFormatTable_columnsAndRows(t *testing.T) {
	diffs := []DiffResult{
		{Rel: "a", Reason: "size changed", Size: 10, Mtime: time.Unix(0, 0)},
		{Rel: "b", Reason: "left only", Size: 0, LeftOnly: true},
	}
	tmp := filepath.Join(t.TempDir(), "out")
	f, _ := os.Create(tmp)
	FormatTable(diffs, f)
	f.Close()
	out, _ := os.ReadFile(tmp)
	if len(out) == 0 {
		t.Fatal("FormatTable produced no output")
	}
	if !bytes.Contains(out, []byte("path")) || !bytes.Contains(out, []byte("size")) {
		t.Error("should contain path and size columns")
	}
	if !bytes.Contains(out, []byte("size changed")) || !bytes.Contains(out, []byte("left only")) {
		t.Error("should contain reasons")
	}
}

func TestFormatTextTree_sortedOutput(t *testing.T) {
	diffs := []DiffResult{
		{Rel: "z/file", Reason: "content differs", Size: 1},
		{Rel: "a/file", Reason: "size changed", Size: 2},
	}
	tmp := filepath.Join(t.TempDir(), "out")
	f, _ := os.Create(tmp)
	FormatTextTree(diffs, f)
	f.Close()
	out, _ := os.ReadFile(tmp)
	aPos := bytes.Index(out, []byte("a/"))
	zPos := bytes.Index(out, []byte("z/"))
	if aPos < 0 || zPos < 0 {
		t.Fatalf("output: %s", out)
	}
	if aPos > zPos {
		t.Error("formatTextTree should sort (a before z)")
	}
}
