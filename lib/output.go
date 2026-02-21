package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FormatTextTree writes diffs as an ASCII tree to w. Case-sensitive sort by path.
func FormatTextTree(diffs []DiffResult, w *os.File) {
	if len(diffs) == 0 {
		return
	}
	sort.Slice(diffs, func(firstDiffIndex, secondDiffIndex int) bool { return diffs[firstDiffIndex].Rel < diffs[secondDiffIndex].Rel })
	seenDirs := make(map[string]bool)
	for _, diff := range diffs {
		parts := strings.Split(filepath.ToSlash(diff.Rel), "/")
		for partIdx := 1; partIdx < len(parts); partIdx++ {
			prefix := strings.Join(parts[:partIdx], "/")
			if !seenDirs[prefix] {
				seenDirs[prefix] = true
				indent := strings.Repeat("  ", partIdx-1)
				fmt.Fprintf(w, "%s%s/\n", indent, parts[partIdx-1])
			}
		}
		indent := strings.Repeat("  ", len(parts)-1)
		name := parts[len(parts)-1]
		mtimeStr := ""
		if !diff.Mtime.IsZero() {
			mtimeStr = diff.Mtime.Format(time.RFC3339)
		}
		line := fmt.Sprintf("%s%s  size=%d  mtime=%s  %s", indent, name, diff.Size, mtimeStr, diff.Reason)
		if diff.Hash != "" {
			line += "  hash=" + diff.Hash
		}
		if diff.LeftOnly {
			line += "  (left only)"
		}
		fmt.Fprintln(w, line)
	}
}

// FormatTable writes diffs as tab-separated columns to w.
func FormatTable(diffs []DiffResult, w *os.File) {
	sort.Slice(diffs, func(firstDiffIndex, secondDiffIndex int) bool { return diffs[firstDiffIndex].Rel < diffs[secondDiffIndex].Rel })
	fmt.Fprintln(w, "path\tsize\tmtime\treason\thash")
	for _, diff := range diffs {
		mtimeStr := ""
		if !diff.Mtime.IsZero() {
			mtimeStr = diff.Mtime.Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", diff.Rel, diff.Size, mtimeStr, diff.Reason, diff.Hash)
	}
}

// FormatJSON writes diffs as JSON array to w.
func FormatJSON(diffs []DiffResult, w *os.File) {
	sort.Slice(diffs, func(firstDiffIndex, secondDiffIndex int) bool { return diffs[firstDiffIndex].Rel < diffs[secondDiffIndex].Rel })
	type item struct {
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		Mtime  string `json:"mtime"`
		Reason string `json:"reason"`
		Hash   string `json:"hash,omitempty"`
	}
	var items []item
	for _, diff := range diffs {
		mtimeStr := ""
		if !diff.Mtime.IsZero() {
			mtimeStr = diff.Mtime.Format(time.RFC3339)
		}
		items = append(items, item{diff.Rel, diff.Size, mtimeStr, diff.Reason, diff.Hash})
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(items)
}

// FormatYAML writes diffs as YAML to w.
func FormatYAML(diffs []DiffResult, w *os.File) {
	sort.Slice(diffs, func(firstDiffIndex, secondDiffIndex int) bool { return diffs[firstDiffIndex].Rel < diffs[secondDiffIndex].Rel })
	type item struct {
		Path   string `yaml:"path"`
		Size   int64  `yaml:"size"`
		Mtime  string `yaml:"mtime"`
		Reason string `yaml:"reason"`
		Hash   string `yaml:"hash,omitempty"`
	}
	var items []item
	for _, diff := range diffs {
		mtimeStr := ""
		if !diff.Mtime.IsZero() {
			mtimeStr = diff.Mtime.Format(time.RFC3339)
		}
		items = append(items, item{diff.Rel, diff.Size, mtimeStr, diff.Reason, diff.Hash})
	}
	encoder := yaml.NewEncoder(w)
	encoder.Encode(items)
}
