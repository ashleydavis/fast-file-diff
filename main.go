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
	startTime := time.Now()
	pool := lib.NewPathPool()
	set := lib.NewDiscoveredSet(pool)
	resultCh := make(chan lib.DiffResult, 256)
	progressCounts := &lib.ProgressCounts{}

	// Phase 1: discover all file pairs by walking both trees.
	walkDoneCh := make(chan struct{})
	go lib.WalkBothTrees(left, right, dirBatchSize, numWorkers, logger, set, walkDoneCh)
	if !quiet && lib.IsTTY(os.Stderr) {
		go discoveryProgressLoop(set, walkDoneCh, numWorkers)
	}
	<-walkDoneCh

	pairPaths := set.PairPaths()
	totalCompared := len(pairPaths)

	// Cheap comparisons (size, mtime) outside workers; only pairs that need hashing go to workers.
	var diffs []lib.DiffResult
	var pairJobs []lib.PairJob
	for _, rel := range pairPaths {
		cached, ok := set.PairCachedInfo(rel)
		if !ok || cached == nil {
			continue
		}
		if cached.LeftSize != cached.RightSize {
			diffs = append(diffs, lib.DiffResult{Rel: rel, Reason: "size changed", Size: cached.LeftSize, Mtime: cached.LeftMtime})
			logger.Log("diff: " + rel + " size changed")
			continue
		}
		if cached.LeftMtime.Equal(cached.RightMtime) {
			continue // same file, no hash needed
		}
		pairJobs = append(pairJobs, lib.PairJob{Rel: rel, Cached: cached})
	}
	progressCounts.TotalPairs = int32(len(pairJobs))

	pairCh := make(chan lib.PairJob, len(pairJobs)+1)
	go func() {
		for _, job := range pairJobs {
			pairCh <- job
		}
		close(pairCh)
	}()
	go lib.RunWorkers(left, right, numWorkers, hashAlg, hashThreshold, pairCh, resultCh, progressCounts)

	compareDoneCh := make(chan struct{})
	if !quiet && lib.IsTTY(os.Stderr) {
		go progressLoop(progressCounts, compareDoneCh, numWorkers)
	}
	for diffResult := range resultCh {
		diffs = append(diffs, diffResult)
		logger.Log("diff: " + diffResult.Rel + " " + diffResult.Reason)
	}
	close(compareDoneCh)
	differentCount := len(diffs)
	sameCount := totalCompared - differentCount
	if sameCount < 0 {
		sameCount = 0
	}
	leftOnlyCount := 0
	for _, rel := range set.LeftOnlyPaths() {
		path := filepath.Join(left, rel)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			diffs = append(diffs, lib.DiffResult{Rel: rel, Reason: "left only", Size: info.Size(), Mtime: info.ModTime().Truncate(time.Second), LeftOnly: true})
			leftOnlyCount++
		}
	}
	rightOnlyCount := 0
	for _, rel := range set.RightOnlyPaths() {
		path := filepath.Join(right, rel)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			diffs = append(diffs, lib.DiffResult{Rel: rel, Reason: "right only", Size: info.Size(), Mtime: info.ModTime().Truncate(time.Second)})
			rightOnlyCount++
		}
	}
	if !quiet {
		elapsed := time.Since(startTime)
		avgPerComparison := time.Duration(0)
		if totalCompared > 0 {
			avgPerComparison = elapsed / time.Duration(totalCompared)
		}
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "Summary:\n")
		fmt.Fprintf(os.Stderr, "  Total files compared:    %d\n", totalCompared)
		fmt.Fprintf(os.Stderr, "  Files only on left:     %d\n", leftOnlyCount)
		fmt.Fprintf(os.Stderr, "  Files only on right:    %d\n", rightOnlyCount)
		fmt.Fprintf(os.Stderr, "  Files different:        %d\n", differentCount)
		fmt.Fprintf(os.Stderr, "  Files same:             %d\n", sameCount)
		fmt.Fprintf(os.Stderr, "  Total time:             %s\n", elapsed.Round(time.Millisecond))
		fmt.Fprintf(os.Stderr, "  Average per comparison: %s\n", avgPerComparison.Round(time.Microsecond))
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

func discoveryProgressLoop(set *lib.DiscoveredSet, doneCh <-chan struct{}, numWorkers int) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-doneCh:
			return
		case <-tick.C:
			n := set.PairsCount()
			fmt.Fprintf(os.Stderr, "\rscanning: %d file pairs found (%d workers)   ", n, numWorkers)
		}
	}
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

func progressLoop(progressCounts *lib.ProgressCounts, doneCh <-chan struct{}, numWorkers int) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-doneCh:
			return
		case <-tick.C:
			processedCount := atomic.LoadInt32(&progressCounts.Processed)
			totalPairs := atomic.LoadInt32(&progressCounts.TotalPairs)
			startTimeNano := atomic.LoadInt64(&progressCounts.StartTimeUnixNano)
			if processedCount == 0 && totalPairs == 0 {
				continue
			}
			if totalPairs > 0 {
				pending := totalPairs - processedCount
				if pending < 0 {
					pending = 0
				}
				remaining := estimateRemainingDuration(processedCount, pending, startTimeNano)
				if remaining > 0 {
					fmt.Fprintf(os.Stderr, "\rcomparing: %d of %d, ~%s remaining (%d workers)   ", processedCount, totalPairs, remaining.Round(time.Second), numWorkers)
				} else {
					fmt.Fprintf(os.Stderr, "\rcomparing: %d of %d (%d workers)   ", processedCount, totalPairs, numWorkers)
				}
			} else {
				enqueuedCount := atomic.LoadInt32(&progressCounts.Enqueued)
				fmt.Fprintf(os.Stderr, "\rprocessed %d, enqueued %d (%d workers)   ", processedCount, enqueuedCount, numWorkers)
			}
		}
	}
}
