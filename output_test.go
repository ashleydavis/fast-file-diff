package main

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
	f, err := os.Create(tmp)
	if err != nil {
		t.Fatal(err)
	}
	formatTable(diffs, f)
	f.Close()
	out, _ := os.ReadFile(tmp)
	if len(out) == 0 {
		t.Fatal("formatTable produced no output")
	}
	if !bytes.Contains(out, []byte("path")) || !bytes.Contains(out, []byte("size")) {
		t.Error("formatTable should contain path and size columns")
	}
	if !bytes.Contains(out, []byte("a")) || !bytes.Contains(out, []byte("b")) {
		t.Error("formatTable should contain both paths")
	}
	if !bytes.Contains(out, []byte("size changed")) || !bytes.Contains(out, []byte("left only")) {
		t.Error("formatTable should contain reasons")
	}
}

func TestFormatTextTree_sortedOutput(t *testing.T) {
	diffs := []DiffResult{
		{Rel: "z/file", Reason: "content differs", Size: 1},
		{Rel: "a/file", Reason: "size changed", Size: 2},
	}
	tmp := filepath.Join(t.TempDir(), "out")
	f, err := os.Create(tmp)
	if err != nil {
		t.Fatal(err)
	}
	formatTextTree(diffs, f)
	f.Close()
	out, _ := os.ReadFile(tmp)
	// Tree output has "a/" and "z/" as dir lines, then "file" under each; sorted order is a before z
	aPos := bytes.Index(out, []byte("a/"))
	zPos := bytes.Index(out, []byte("z/"))
	if aPos < 0 || zPos < 0 {
		t.Fatalf("output should contain both path segments: %s", out)
	}
	if aPos > zPos {
		t.Error("formatTextTree should sort by path (a before z)")
	}
}
