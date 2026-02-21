package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
