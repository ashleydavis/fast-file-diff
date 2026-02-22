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
	sort.Slice(diffs, func(firstDiffIndex, secondDiffIndex int) bool {
		return diffs[firstDiffIndex].Rel < diffs[secondDiffIndex].Rel
	})
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
		sizeStr := fmt.Sprintf("%d", diff.LeftSize)
		if !diff.LeftOnly && (diff.RightSize != 0 || diff.LeftSize != diff.RightSize) {
			sizeStr = fmt.Sprintf("%d/%d", diff.LeftSize, diff.RightSize)
		}
		mtimeStr := ""
		if !diff.LeftMtime.IsZero() {
			mtimeStr = diff.LeftMtime.Format(time.RFC3339)
			if !diff.LeftOnly && !diff.RightMtime.IsZero() && diff.LeftMtime != diff.RightMtime {
				mtimeStr += "/" + diff.RightMtime.Format(time.RFC3339)
			}
		}
		line := fmt.Sprintf("%s%s  size=%s  mtime=%s  %s", indent, name, sizeStr, mtimeStr, diff.Reason)
		if diff.LeftHash != "" || diff.RightHash != "" {
			line += "  hash=" + diff.LeftHash
			if diff.RightHash != "" {
				line += "/" + diff.RightHash
			}
		}
		if diff.LeftOnly {
			line += "  (left only)"
		}
		fmt.Fprintln(w, line)
	}
}

// FormatTable writes diffs as tab-separated columns to w.
func FormatTable(diffs []DiffResult, w *os.File) {
	sort.Slice(diffs, func(firstDiffIndex, secondDiffIndex int) bool {
		return diffs[firstDiffIndex].Rel < diffs[secondDiffIndex].Rel
	})
	fmt.Fprintln(w, "path\tleft_size\tright_size\tleft_mtime\tright_mtime\treason\tleft_hash\tright_hash")
	for _, diff := range diffs {
		leftMtimeStr := ""
		if !diff.LeftMtime.IsZero() {
			leftMtimeStr = diff.LeftMtime.Format(time.RFC3339)
		}
		rightMtimeStr := ""
		if !diff.RightMtime.IsZero() {
			rightMtimeStr = diff.RightMtime.Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\n", diff.Rel, diff.LeftSize, diff.RightSize, leftMtimeStr, rightMtimeStr, diff.Reason, diff.LeftHash, diff.RightHash)
	}
}

// FormatJSON writes diffs as JSON array to w.
func FormatJSON(diffs []DiffResult, w *os.File) {
	sort.Slice(diffs, func(firstDiffIndex, secondDiffIndex int) bool {
		return diffs[firstDiffIndex].Rel < diffs[secondDiffIndex].Rel
	})
	type item struct {
		Path       string `json:"path"`
		LeftSize   int64  `json:"left_size"`
		RightSize  int64  `json:"right_size"`
		LeftMtime  string `json:"left_mtime"`
		RightMtime string `json:"right_mtime"`
		Reason     string `json:"reason"`
		LeftHash   string `json:"left_hash,omitempty"`
		RightHash  string `json:"right_hash,omitempty"`
	}
	var items []item
	for _, diff := range diffs {
		leftMtimeStr := ""
		if !diff.LeftMtime.IsZero() {
			leftMtimeStr = diff.LeftMtime.Format(time.RFC3339)
		}
		rightMtimeStr := ""
		if !diff.RightMtime.IsZero() {
			rightMtimeStr = diff.RightMtime.Format(time.RFC3339)
		}
		items = append(items, item{diff.Rel, diff.LeftSize, diff.RightSize, leftMtimeStr, rightMtimeStr, diff.Reason, diff.LeftHash, diff.RightHash})
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(items)
}

// FormatYAML writes diffs as YAML to w.
func FormatYAML(diffs []DiffResult, w *os.File) {
	sort.Slice(diffs, func(firstDiffIndex, secondDiffIndex int) bool {
		return diffs[firstDiffIndex].Rel < diffs[secondDiffIndex].Rel
	})
	type item struct {
		Path       string `yaml:"path"`
		LeftSize   int64  `yaml:"left_size"`
		RightSize  int64  `yaml:"right_size"`
		LeftMtime  string `yaml:"left_mtime"`
		RightMtime string `yaml:"right_mtime"`
		Reason     string `yaml:"reason"`
		LeftHash   string `yaml:"left_hash,omitempty"`
		RightHash  string `yaml:"right_hash,omitempty"`
	}
	var items []item
	for _, diff := range diffs {
		leftMtimeStr := ""
		if !diff.LeftMtime.IsZero() {
			leftMtimeStr = diff.LeftMtime.Format(time.RFC3339)
		}
		rightMtimeStr := ""
		if !diff.RightMtime.IsZero() {
			rightMtimeStr = diff.RightMtime.Format(time.RFC3339)
		}
		items = append(items, item{diff.Rel, diff.LeftSize, diff.RightSize, leftMtimeStr, rightMtimeStr, diff.Reason, diff.LeftHash, diff.RightHash})
	}
	encoder := yaml.NewEncoder(w)
	encoder.Encode(items)
}
