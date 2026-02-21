package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/photosphere/fast-file-diff-go/lib"
	"github.com/spf13/cobra"
)

const pairQueueCap = 10000

const (
	ExitSuccess  = 0
	ExitUsage    = 1
	ExitFatal    = 2
	ExitNonFatal = 3
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
	if err := lib.EnsureDir(left); err != nil {
		fmt.Fprintf(os.Stderr, "left directory: %v\n", err)
		os.Exit(ExitFatal)
	}
	if err := lib.EnsureDir(right); err != nil {
		fmt.Fprintf(os.Stderr, "right directory: %v\n", err)
		os.Exit(ExitFatal)
	}
	logger, err := lib.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(ExitFatal)
	}
	defer logger.Close()
	if !quiet {
		defer logger.PrintLogPaths()
	}
	logger.Log("started comparison")
	pool := lib.NewPathPool()
	set := lib.NewDiscoveredSet(pool)
	pairCh := make(chan string, pairQueueCap)
	resultCh := make(chan lib.DiffResult, 256)
	progressCounts := &lib.ProgressCounts{}
	go lib.WalkBothTrees(left, right, dirBatchSize, logger, set, pairCh)
	lib.RunWorkers(left, right, numWorkers, hashAlg, hashThreshold, pairCh, resultCh, progressCounts)
	doneCh := make(chan struct{})
	if !quiet && lib.IsTTY(os.Stderr) {
		go progressLoop(progressCounts, doneCh)
	}
	var diffs []lib.DiffResult
	for diffResult := range resultCh {
		diffs = append(diffs, diffResult)
		logger.Log("diff: " + diffResult.Rel + " " + diffResult.Reason)
	}
	close(doneCh)
	for _, rel := range set.LeftOnlyPaths() {
		path := filepath.Join(left, rel)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			diffs = append(diffs, lib.DiffResult{Rel: rel, Reason: "left only", Size: info.Size(), Mtime: info.ModTime().Truncate(time.Second), LeftOnly: true})
		}
	}
	for _, rel := range set.RightOnlyPaths() {
		path := filepath.Join(right, rel)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			diffs = append(diffs, lib.DiffResult{Rel: rel, Reason: "right only", Size: info.Size(), Mtime: info.ModTime().Truncate(time.Second)})
		}
	}
	switch outputFormat {
	case "table":
		lib.FormatTable(diffs, os.Stdout)
	case "json":
		lib.FormatJSON(diffs, os.Stdout)
	case "yaml":
		lib.FormatYAML(diffs, os.Stdout)
	default:
		lib.FormatTextTree(diffs, os.Stdout)
	}
	if logger.NonFatalCount() > 0 {
		if !quiet {
			fmt.Fprintln(os.Stderr, "Errors occurred; check the error log for details.")
		}
		os.Exit(ExitNonFatal)
	}
	return nil
}

func estimateRemainingFromElapsed(elapsed time.Duration, processed, pending int32) time.Duration {
	if processed <= 0 || pending <= 0 {
		return 0
	}
	averagePerPair := elapsed / time.Duration(processed)
	return averagePerPair * time.Duration(pending)
}

func estimateRemainingDuration(processed, pending int32, startTimeUnixNano int64) time.Duration {
	if startTimeUnixNano == 0 {
		return 0
	}
	elapsed := time.Since(time.Unix(0, startTimeUnixNano))
	return estimateRemainingFromElapsed(elapsed, processed, pending)
}

func progressLoop(progressCounts *lib.ProgressCounts, doneCh <-chan struct{}) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-doneCh:
			return
		case <-tick.C:
			processedCount := atomic.LoadInt32(&progressCounts.Processed)
			enqueuedCount := atomic.LoadInt32(&progressCounts.Enqueued)
			pending := enqueuedCount - processedCount
			if enqueuedCount == 0 && processedCount == 0 {
				continue
			}
			startTimeNano := atomic.LoadInt64(&progressCounts.StartTimeUnixNano)
			remaining := estimateRemainingDuration(processedCount, pending, startTimeNano)
			if remaining > 0 {
				fmt.Fprintf(os.Stderr, "\rprocessed %d, pending %d, ~%s remaining   ", processedCount, pending, remaining.Round(time.Second))
			} else {
				fmt.Fprintf(os.Stderr, "\rprocessed %d, pending %d   ", processedCount, pending)
			}
		}
	}
}
