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
	outFile, _ := os.Create(tmp)
	FormatTable(diffs, outFile)
	outFile.Close()
	output, _ := os.ReadFile(tmp)
	if len(output) == 0 {
		t.Fatal("FormatTable produced no output")
	}
	if !bytes.Contains(output, []byte("path")) || !bytes.Contains(output, []byte("size")) {
		t.Error("should contain path and size columns")
	}
	if !bytes.Contains(output, []byte("size changed")) || !bytes.Contains(output, []byte("left only")) {
		t.Error("should contain reasons")
	}
}

func TestFormatTextTree_sortedOutput(t *testing.T) {
	diffs := []DiffResult{
		{Rel: "z/file", Reason: "content differs", Size: 1},
		{Rel: "a/file", Reason: "size changed", Size: 2},
	}
	tmp := filepath.Join(t.TempDir(), "out")
	outFile, _ := os.Create(tmp)
	FormatTextTree(diffs, outFile)
	outFile.Close()
	output, _ := os.ReadFile(tmp)
	aPos := bytes.Index(output, []byte("a/"))
	zPos := bytes.Index(output, []byte("z/"))
	if aPos < 0 || zPos < 0 {
		t.Fatalf("output: %s", output)
	}
	if aPos > zPos {
		t.Error("formatTextTree should sort (a before z)")
	}
}
