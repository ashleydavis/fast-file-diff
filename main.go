package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

const pairQueueCap = 10000

// Exit code constants per SPEC (CLI / help).
const (
	ExitSuccess   = 0
	ExitUsage     = 1
	ExitFatal     = 2
	ExitNonFatal  = 3
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(ExitUsage)
	}
}

var dirBatchSize int
var numWorkers int
var hashAlg string
var hashThreshold int
var outputFormat string
var quiet bool

var rootCmd = &cobra.Command{
	Use:   "ffd <left-dir> <right-dir>",
	Short: "Fast file diff between two directory trees",
	Long:  "Compare two directory trees recursively. Left dir and right dir are required positional arguments.",
	Args:  cobra.MatchAll(cobra.ArbitraryArgs, requireZeroOrTwoArgs),
	RunE:  runRoot,
}

func init() {
	rootCmd.Flags().IntVar(&dirBatchSize, "dir-batch-size", 4096, "On Linux: batch size for directory reads (entries per syscall)")
	rootCmd.Flags().IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of worker goroutines for comparing file pairs")
	rootCmd.Flags().StringVar(&hashAlg, "hash", "xxhash", "Hash algorithm for content comparison: xxhash, sha256, md5")
	rootCmd.Flags().IntVar(&hashThreshold, "threshold", 10*1024*1024, "Size threshold in bytes: files smaller are read in full to hash, larger are streamed")
	rootCmd.Flags().StringVar(&outputFormat, "format", "text", "Output format: text, table, json, yaml")
	rootCmd.Flags().BoolVar(&quiet, "quiet", false, "Suppress progress and final error-log message (for scripting)")
}

func requireZeroOrTwoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || len(args) == 2 {
		return nil
	}
	return fmt.Errorf("requires 0 or 2 arguments, got %d", len(args))
}

func runRoot(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		cmd.SetOut(os.Stdout)
		return cmd.Usage()
	}
	left, right := args[0], args[1]
	if err := ensureDir(left); err != nil {
		fmt.Fprintf(os.Stderr, "left directory: %v\n", err)
		os.Exit(ExitFatal)
	}
	if err := ensureDir(right); err != nil {
		fmt.Fprintf(os.Stderr, "right directory: %v\n", err)
		os.Exit(ExitFatal)
	}
	logger, err := NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(ExitFatal)
	}
	defer logger.Close()
	if !quiet {
		defer logger.PrintLogPaths()
	}
	logger.Log("started comparison")
	pool := newPathPool()
	set := newDiscoveredSet(pool)
	pairCh := make(chan string, pairQueueCap)
	resultCh := make(chan DiffResult, 256)
	progress := &progressCounts{}
	go walkBothTrees(left, right, dirBatchSize, logger, set, pairCh)
	runWorkers(left, right, numWorkers, hashAlg, hashThreshold, pairCh, resultCh, progress)
	doneCh := make(chan struct{})
	if !quiet && isTTY(os.Stderr) {
		go progressLoop(progress, doneCh)
	}
	var diffs []DiffResult
	for diffResult := range resultCh {
		diffs = append(diffs, diffResult)
		logger.Log("diff: " + diffResult.Rel + " " + diffResult.Reason)
	}
	close(doneCh)
	for _, rel := range set.LeftOnlyPaths() {
		path := filepath.Join(left, rel)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			diffs = append(diffs, DiffResult{Rel: rel, Reason: "left only", Size: info.Size(), Mtime: info.ModTime().Truncate(time.Second), LeftOnly: true})
		}
	}
	for _, rel := range set.RightOnlyPaths() {
		path := filepath.Join(right, rel)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			diffs = append(diffs, DiffResult{Rel: rel, Reason: "right only", Size: info.Size(), Mtime: info.ModTime().Truncate(time.Second)})
		}
	}
	switch outputFormat {
	case "table":
		formatTable(diffs, os.Stdout)
	case "json":
		formatJSON(diffs, os.Stdout)
	case "yaml":
		formatYAML(diffs, os.Stdout)
	default:
		formatTextTree(diffs, os.Stdout)
	}
	if logger.NonFatalCount() > 0 {
		if !quiet {
			fmt.Fprintln(os.Stderr, "Errors occurred; check the error log for details.")
		}
		os.Exit(ExitNonFatal)
	}
	return nil
}

// estimateRemainingFromElapsed returns estimated remaining time from elapsed duration and processed/pending counts.
// Used for testing; the progress loop uses estimateRemainingDuration which derives elapsed from start time.
func estimateRemainingFromElapsed(elapsed time.Duration, processed, pending int32) time.Duration {
	if processed <= 0 || pending <= 0 {
		return 0
	}
	averagePerPair := elapsed / time.Duration(processed)
	return averagePerPair * time.Duration(pending)
}

// estimateRemainingDuration returns an estimate of time remaining based on processed count,
// pending count, and start time. Returns zero if not enough data (processed <= 0, pending <= 0, or start not set).
func estimateRemainingDuration(processed, pending int32, startTimeUnixNano int64) time.Duration {
	if startTimeUnixNano == 0 {
		return 0
	}
	elapsed := time.Since(time.Unix(0, startTimeUnixNano))
	return estimateRemainingFromElapsed(elapsed, processed, pending)
}

func progressLoop(p *progressCounts, doneCh <-chan struct{}) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-doneCh:
			return
		case <-tick.C:
			proc := atomic.LoadInt32(&p.processed)
			enq := atomic.LoadInt32(&p.enqueued)
			pending := enq - proc
			if enq == 0 && proc == 0 {
				continue
			}
			startNano := atomic.LoadInt64(&p.startTimeUnixNano)
			remaining := estimateRemainingDuration(proc, pending, startNano)
			if remaining > 0 {
				fmt.Fprintf(os.Stderr, "\rprocessed %d, pending %d, ~%s remaining   ", proc, pending, remaining.Round(time.Second))
			} else {
				fmt.Fprintf(os.Stderr, "\rprocessed %d, pending %d   ", proc, pending)
			}
		}
	}
}

// ensureDir returns nil if path is an existing directory; otherwise an error.
func ensureDir(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}
	return nil
}
