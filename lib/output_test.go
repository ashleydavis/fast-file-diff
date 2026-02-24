package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatTable_columnsAndRows(t *testing.T) {
	diffs := []DiffResult{
		{Rel: "a", Reason: "size changed", LeftSize: 10, RightSize: 20, LeftMtime: time.Unix(0, 0), RightMtime: time.Unix(0, 0)},
		{Rel: "b", Reason: "left only", LeftOnly: true},
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
		{Rel: "z/file", Reason: "content differs", LeftSize: 1, RightSize: 1, LeftHash: "a", RightHash: "b"},
		{Rel: "a/file", Reason: "size changed", LeftSize: 2, RightSize: 3},
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

func TestFormatJSON_validJSONAndContainsPath(t *testing.T) {
	diffs := []DiffResult{
		{Rel: "x", Reason: "size changed", LeftSize: 1, RightSize: 2, LeftMtime: time.Unix(0, 0), RightMtime: time.Unix(0, 0)},
	}
	tmp := filepath.Join(t.TempDir(), "out")
	outFile, err := os.Create(tmp)
	if err != nil {
		t.Fatal(err)
	}
	FormatJSON(diffs, outFile)
	outFile.Close()
	output, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]interface{}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("FormatJSON produced invalid JSON: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 item, got %d", len(decoded))
	}
	if decoded[0]["path"] != "x" || decoded[0]["reason"] != "size changed" {
		t.Errorf("decoded[0] = %v", decoded[0])
	}
}

func TestFormatYAML_containsPathAndReason(t *testing.T) {
	diffs := []DiffResult{
		{Rel: "y", Reason: "content differs", LeftSize: 1, RightSize: 1, LeftHash: "a", RightHash: "b"},
	}
	tmp := filepath.Join(t.TempDir(), "out")
	outFile, err := os.Create(tmp)
	if err != nil {
		t.Fatal(err)
	}
	FormatYAML(diffs, outFile)
	outFile.Close()
	output, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	str := string(output)
	if !strings.Contains(str, "path:") || !strings.Contains(str, "y") || !strings.Contains(str, "content differs") {
		t.Errorf("FormatYAML output missing path or reason: %s", str)
	}
}
