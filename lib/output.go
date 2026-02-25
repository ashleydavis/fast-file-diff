package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// formatTextTreeBody writes a single tree of diffs to writer. Caller must pass already-sorted diffs.
func formatTextTreeBody(diffs []DiffResult, writer io.Writer) {
	if len(diffs) == 0 {
		return
	}
	// First pass: build rows and measure column widths for alignment
	type row struct {
		dirLines   []string // directory header lines to print before this file
		namePart   string   // indent + filename
		sizePart   string   // size=...
		mtimePart  string   // mtime=...
		reasonPart string   // reason and optional hash
	}
	var rows []row
	seenDirs := make(map[string]bool)
	maxNameLen := 0
	maxSizeLen := 0
	maxMtimeLen := 0
	for _, diff := range diffs {
		parts := strings.Split(filepath.ToSlash(diff.Rel), "/")
		var dirLines []string
		for partIdx := 1; partIdx < len(parts); partIdx++ {
			prefix := strings.Join(parts[:partIdx], "/")
			if !seenDirs[prefix] {
				seenDirs[prefix] = true
				indent := strings.Repeat("  ", partIdx-1)
				dirLines = append(dirLines, indent+parts[partIdx-1]+"/")
			}
		}
		indent := strings.Repeat("  ", len(parts)-1)
		name := parts[len(parts)-1]
		namePart := indent + name
		var sizeStr string
		if diff.LeftOnly {
			sizeStr = fmt.Sprintf("%d", diff.LeftSize)
		} else if diff.Reason == "right only" {
			sizeStr = fmt.Sprintf("%d", diff.RightSize)
		} else if diff.RightSize != 0 || diff.LeftSize != diff.RightSize {
			sizeStr = fmt.Sprintf("%d/%d", diff.LeftSize, diff.RightSize)
		} else {
			sizeStr = fmt.Sprintf("%d", diff.LeftSize)
		}
		sizePart := "size=" + sizeStr
		mtimeStr := ""
		if diff.LeftOnly && !diff.LeftMtime.IsZero() {
			mtimeStr = diff.LeftMtime.Format(time.RFC3339)
		} else if diff.Reason == "right only" && !diff.RightMtime.IsZero() {
			mtimeStr = diff.RightMtime.Format(time.RFC3339)
		} else if !diff.LeftMtime.IsZero() {
			mtimeStr = diff.LeftMtime.Format(time.RFC3339)
			if !diff.RightMtime.IsZero() && diff.LeftMtime != diff.RightMtime {
				mtimeStr += "/" + diff.RightMtime.Format(time.RFC3339)
			}
		}
		mtimePart := "mtime=" + mtimeStr
		reasonPart := diff.Reason
		if diff.LeftHash != "" || diff.RightHash != "" {
			reasonPart += "  hash=" + diff.LeftHash
			if diff.RightHash != "" {
				reasonPart += "/" + diff.RightHash
			}
		}
		if n := len(namePart); n > maxNameLen {
			maxNameLen = n
		}
		if n := len(sizePart); n > maxSizeLen {
			maxSizeLen = n
		}
		if n := len(mtimePart); n > maxMtimeLen {
			maxMtimeLen = n
		}
		rows = append(rows, row{dirLines, namePart, sizePart, mtimePart, reasonPart})
	}
	// Second pass: print directory lines and aligned file lines
	for _, row := range rows {
		for _, line := range row.dirLines {
			fmt.Fprintln(writer, line)
		}
		fmt.Fprintf(writer, "%-*s  %-*s  %-*s  %s\n", maxNameLen, row.namePart, maxSizeLen, row.sizePart, maxMtimeLen, row.mtimePart, row.reasonPart)
	}
}

// FormatTextTree writes diffs as an ASCII tree to writer with aligned columns. Case-sensitive sort by path.
func FormatTextTree(diffs []DiffResult, writer io.Writer) {
	if len(diffs) == 0 {
		return
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Rel < diffs[j].Rel })
	formatTextTreeBody(diffs, writer)
}

// FormatTextTreeWithSections splits diffs into different, left-only, and right-only, builds sameDiffs from compareResults when showSame, then writes four sections (Different, Same (identical), Left only, Right only) to stdout and the logger, each with a header and a tree.
func FormatTextTreeWithSections(diffs []DiffResult, differentCount int, compareResults []CompareResult, showSame bool) {
	log := Log
	var out strings.Builder

	differentDiffs := diffs[:differentCount]
	var leftOnlyDiffs, rightOnlyDiffs []DiffResult
	for i := differentCount; i < len(diffs); i++ {
		if diffs[i].LeftOnly {
			leftOnlyDiffs = append(leftOnlyDiffs, diffs[i])
		} else {
			rightOnlyDiffs = append(rightOnlyDiffs, diffs[i])
		}
	}
	var sameDiffs []DiffResult
	if showSame {
		for _, result := range compareResults {
			if result.Diff == nil {
				sameDiffs = append(sameDiffs, DiffResult{Rel: result.RelativePath, Reason: result.Reason})
			}
		}
	}
	if len(differentDiffs) > 0 {
		sort.Slice(differentDiffs, func(i, j int) bool { return differentDiffs[i].Rel < differentDiffs[j].Rel })
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "Different:")
		formatTextTreeBody(differentDiffs, &out)
		fmt.Fprintln(&out)
	}
	if showSame && len(sameDiffs) > 0 {
		sort.Slice(sameDiffs, func(i, j int) bool { return sameDiffs[i].Rel < sameDiffs[j].Rel })
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "Same (identical):")
		formatTextTreeBody(sameDiffs, &out)
		fmt.Fprintln(&out)
	}
	if len(leftOnlyDiffs) > 0 {
		sort.Slice(leftOnlyDiffs, func(i, j int) bool { return leftOnlyDiffs[i].Rel < leftOnlyDiffs[j].Rel })
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "Left only:")
		formatTextTreeBody(leftOnlyDiffs, &out)
		fmt.Fprintln(&out)
	}
	if len(rightOnlyDiffs) > 0 {
		sort.Slice(rightOnlyDiffs, func(i, j int) bool { return rightOnlyDiffs[i].Rel < rightOnlyDiffs[j].Rel })
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "Right only:")
		formatTextTreeBody(rightOnlyDiffs, &out)
	}
	outputText := out.String()
	os.Stdout.WriteString(outputText)
	if log != nil && outputText != "" {
		log.Write(outputText)
	}
}

// FormatTable writes diffs as tab-separated columns to stdout and the logger.
func FormatTable(diffs []DiffResult) {
	log := Log
	var out strings.Builder
	formatTableTo(diffs, &out)
	outputText := out.String()
	os.Stdout.WriteString(outputText)
	if log != nil && outputText != "" {
		log.Write(outputText)
	}
}

// formatTableTo writes tab-separated columns to writer (for tests and FormatTable).
func formatTableTo(diffs []DiffResult, writer io.Writer) {
	sort.Slice(diffs, func(firstDiffIndex, secondDiffIndex int) bool {
		return diffs[firstDiffIndex].Rel < diffs[secondDiffIndex].Rel
	})
	fmt.Fprintln(writer, "path\tleft_size\tright_size\tleft_mtime\tright_mtime\treason\tleft_hash\tright_hash")
	for _, diff := range diffs {
		leftMtimeStr := ""
		if !diff.LeftMtime.IsZero() {
			leftMtimeStr = diff.LeftMtime.Format(time.RFC3339)
		}
		rightMtimeStr := ""
		if !diff.RightMtime.IsZero() {
			rightMtimeStr = diff.RightMtime.Format(time.RFC3339)
		}
		fmt.Fprintf(writer, "%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\n", filepath.ToSlash(diff.Rel), diff.LeftSize, diff.RightSize, leftMtimeStr, rightMtimeStr, diff.Reason, diff.LeftHash, diff.RightHash)
	}
}

// FormatJSON writes diffs as JSON array to stdout and the logger.
func FormatJSON(diffs []DiffResult) {
	log := Log
	var buf bytes.Buffer
	formatJSONTo(diffs, &buf)
	outputText := buf.String()
	os.Stdout.WriteString(outputText)
	if log != nil && outputText != "" {
		log.Write(outputText)
	}
}

// formatJSONTo writes JSON array to writer (for tests and FormatJSON).
func formatJSONTo(diffs []DiffResult, writer io.Writer) {
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
		items = append(items, item{filepath.ToSlash(diff.Rel), diff.LeftSize, diff.RightSize, leftMtimeStr, rightMtimeStr, diff.Reason, diff.LeftHash, diff.RightHash})
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.Encode(items)
}

// FormatYAML writes diffs as YAML to stdout and the logger.
func FormatYAML(diffs []DiffResult) {
	log := Log
	var buf bytes.Buffer
	formatYAMLTo(diffs, &buf)
	outputText := buf.String()
	os.Stdout.WriteString(outputText)
	if log != nil && outputText != "" {
		log.Write(outputText)
	}
}

// formatYAMLTo writes YAML to writer (for tests and FormatYAML).
func formatYAMLTo(diffs []DiffResult, writer io.Writer) {
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
		items = append(items, item{filepath.ToSlash(diff.Rel), diff.LeftSize, diff.RightSize, leftMtimeStr, rightMtimeStr, diff.Reason, diff.LeftHash, diff.RightHash})
	}
	encoder := yaml.NewEncoder(writer)
	encoder.Encode(items)
}
