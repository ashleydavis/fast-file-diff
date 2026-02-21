package main

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

// formatTextTree writes diffs as an ASCII tree to w. Case-sensitive sort by path.
func formatTextTree(diffs []DiffResult, w *os.File) {
	if len(diffs) == 0 {
		return
	}
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Rel < diffs[j].Rel
	})
	seenDirs := make(map[string]bool)
	for _, d := range diffs {
		parts := strings.Split(filepath.ToSlash(d.Rel), "/")
		for i := 1; i < len(parts); i++ {
			prefix := strings.Join(parts[:i], "/")
			if !seenDirs[prefix] {
				seenDirs[prefix] = true
				indent := strings.Repeat("  ", i-1)
				fmt.Fprintf(w, "%s%s/\n", indent, parts[i-1])
			}
		}
		indent := strings.Repeat("  ", len(parts)-1)
		name := parts[len(parts)-1]
		mt := ""
		if !d.Mtime.IsZero() {
			mt = d.Mtime.Format(time.RFC3339)
		}
		line := fmt.Sprintf("%s%s  size=%d  mtime=%s  %s", indent, name, d.Size, mt, d.Reason)
		if d.Hash != "" {
			line += "  hash=" + d.Hash
		}
		if d.LeftOnly {
			line += "  (left only)"
		}
		fmt.Fprintln(w, line)
	}
}

func formatTable(diffs []DiffResult, w *os.File) {
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Rel < diffs[j].Rel })
	fmt.Fprintln(w, "path\tsize\tmtime\treason\thash")
	for _, d := range diffs {
		mt := ""
		if !d.Mtime.IsZero() {
			mt = d.Mtime.Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", d.Rel, d.Size, mt, d.Reason, d.Hash)
	}
}

func formatJSON(diffs []DiffResult, w *os.File) {
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Rel < diffs[j].Rel })
	type item struct {
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		Mtime  string `json:"mtime"`
		Reason string `json:"reason"`
		Hash   string `json:"hash,omitempty"`
	}
	var items []item
	for _, d := range diffs {
		mt := ""
		if !d.Mtime.IsZero() {
			mt = d.Mtime.Format(time.RFC3339)
		}
		items = append(items, item{d.Rel, d.Size, mt, d.Reason, d.Hash})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(items)
}

func formatYAML(diffs []DiffResult, w *os.File) {
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Rel < diffs[j].Rel })
	type item struct {
		Path   string `yaml:"path"`
		Size   int64  `yaml:"size"`
		Mtime  string `yaml:"mtime"`
		Reason string `yaml:"reason"`
		Hash   string `yaml:"hash,omitempty"`
	}
	var items []item
	for _, d := range diffs {
		mt := ""
		if !d.Mtime.IsZero() {
			mt = d.Mtime.Format(time.RFC3339)
		}
		items = append(items, item{d.Rel, d.Size, mt, d.Reason, d.Hash})
	}
	enc := yaml.NewEncoder(w)
	enc.Encode(items)
}
