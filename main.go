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

// runSummary holds the counts and timings needed to display the run summary.
type runSummary struct {
	leftDir                  string
	rightDir                 string
	totalCompared            int
	leftOnlyCount            int
	rightOnlyCount           int
	differentCount           int
	sameCount                int
	startTime                time.Time
	scanDuration             time.Duration
	compareDuration          time.Duration
	workerUtilizationPercent int
}

// displaySummary writes the run summary to the logger (always) and to stderr when printToStderr is true.
func displaySummary(logger *lib.Logger, printToStderr bool, s runSummary) {
	elapsed := time.Since(s.startTime)
	avgPerComparison := time.Duration(0)
	if s.totalCompared > 0 {
		avgPerComparison = elapsed / time.Duration(s.totalCompared)
	}
	lines := []string{
		"",
		"Summary:",
		fmt.Sprintf("  Left directory:          %s", s.leftDir),
		fmt.Sprintf("  Right directory:         %s", s.rightDir),
		fmt.Sprintf("  Total files compared:   %d", s.totalCompared),
		fmt.Sprintf("  Files only on left:     %d", s.leftOnlyCount),
		fmt.Sprintf("  Files only on right:    %d", s.rightOnlyCount),
		fmt.Sprintf("  Files different:        %d", s.differentCount),
		fmt.Sprintf("  Files same:             %d", s.sameCount),
		fmt.Sprintf("  Scanning:               %s", s.scanDuration.Round(time.Millisecond)),
		fmt.Sprintf("  Comparing:              %s", s.compareDuration.Round(time.Millisecond)),
		fmt.Sprintf("  Total time:             %s", elapsed.Round(time.Millisecond)),
		fmt.Sprintf("  Average per comparison: %s", avgPerComparison.Round(time.Microsecond)),
		fmt.Sprintf("  Workers utilized:       %d%%", s.workerUtilizationPercent),
	}
	for _, line := range lines {
		if line == "" {
			logger.Log("")
			if printToStderr {
				fmt.Fprintln(os.Stderr)
			}
		} else {
			logger.Log(line)
			if printToStderr {
				fmt.Fprintln(os.Stderr, line)
			}
		}
	}
}

// Runs the CLI; on any error exits with ExitUsage so scripts get a consistent exit code.
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(ExitUsage)
	}
}

// Version is set at build time via -ldflags "-X main.Version=..."; empty means "dev".
var Version string

// Hold flag values so runRoot can read them without passing through cobra.
var dirBatchSize int
var numWorkers int
var hashAlg string
var hashThreshold int
var outputFormat string
var quiet bool

// Single top-level command; requireZeroOrTwoArgs validates args, runRoot does the diff.
var rootCmd = &cobra.Command{
	Use:   "ffd <left-dir> <right-dir>",
	Short: "Fast file diff between two directory trees",
	Long:  "Compare two directory trees recursively. Left dir and right dir are required positional arguments.",
	Args:  cobra.MatchAll(cobra.ArbitraryArgs, requireZeroOrTwoArgs),
	RunE:  runRoot,
}

// Binds flags to the package-level vars; defaults match the spec (e.g. xxhash, 10MiB threshold).
func init() {
	if Version == "" {
		Version = "dev"
	}
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.Flags().IntVar(&dirBatchSize, "dir-batch-size", 4096, "Batch size for directory reads (entries per syscall)")
	rootCmd.Flags().IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of worker goroutines for comparing file pairs")
	rootCmd.Flags().StringVar(&hashAlg, "hash", "xxhash", "Hash algorithm for content comparison: xxhash, sha256, md5")
	rootCmd.Flags().IntVar(&hashThreshold, "threshold", 10*1024*1024, "Size threshold in bytes: files smaller are read in full to hash, larger are streamed")
	rootCmd.Flags().StringVar(&outputFormat, "format", "text", "Output format: text, table, json, yaml")
	rootCmd.Flags().BoolVar(&quiet, "quiet", false, "Suppress progress and final error-log message (for scripting)")
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(versionCmd)
}

// lsCmd lists all files under a directory recursively (one relative path per line). Uses the same walk code as the diff.
var lsCmd = &cobra.Command{
	Use:   "ls [directory]",
	Short: "List files in a directory recursively",
	Long:  "Walk the given directory and print the relative path of every file (one per line). Uses the same walk implementation as the diff.",
	Args:  cobra.ExactArgs(1),
	RunE:  runLs,
}

// versionCmd prints the version number to stdout and exits (script-friendly).
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  "Print the version number of this build to stdout. Use --version for the same from the root command.",
	Args:  cobra.NoArgs,
	RunE:  runVersion,
}

// runVersion writes the version string to stdout; used by the version command.
func runVersion(cmd *cobra.Command, args []string) error {
	fmt.Println(Version)
	return nil
}

// Enforces 0 args (for --help) or 2 args (left and right dir); used as cobra's Args so users get a clear error.
func requireZeroOrTwoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || len(args) == 2 {
		return nil
	}
	return fmt.Errorf("requires 0 or 2 arguments, got %d", len(args))
}

// runLs walks the given directory and prints each file's relative path to stdout (one per line).
func runLs(cmd *cobra.Command, args []string) error {
	root := args[0]
	if err := lib.EnsureDir(root); err != nil {
		return fmt.Errorf("not a directory: %w", err)
	}
	start := time.Now()
	var count int
	lib.WalkTree(root, func(rel string, isDir bool) {
		if !isDir {
			count++
			fmt.Fprintln(cmd.OutOrStdout(), rel)
		}
	})
	elapsed := time.Since(start)
	fmt.Fprintf(cmd.ErrOrStderr(), "Listed %d files in %v\n", count, elapsed.Round(time.Millisecond))
	return nil
}

// Validates dirs, walks both trees, compares pairs (with progress when not quiet), then writes diffs in the chosen format. Drives lib for walk, discovery, hashing, and output; progress and logging stay here so the CLI controls UX.
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
	// Print compared directories at start (logger always; stderr when not quiet).
	logger.Log("left directory: " + left)
	logger.Log("right directory: " + right)
	if !quiet {
		fmt.Fprintln(os.Stderr, "Left directory:  ", left)
		fmt.Fprintln(os.Stderr, "Right directory: ", right)
	}
	logger.Log("started comparison")
	startTime := time.Now()
	pool := lib.NewPathPool()
	set := lib.NewDiscoveredSet(pool)
	compareResultCh := make(chan lib.CompareResult, 256)
	progressCounts := &lib.ProgressCounts{}

	// Phase 1: discover all file pairs by walking both trees.
	walkDoneCh := make(chan struct{})
	const utilWindowTicks = 30 // ~3 seconds at 100ms tick; longer window so "workers active" is meaningful when work is bursty
	walkWorkerUtilization := lib.NewWorkerUtilization(numWorkers, utilWindowTicks)
	go lib.WalkBothTrees(left, right, dirBatchSize, numWorkers, logger, set, walkDoneCh, walkWorkerUtilization)
	if !quiet && lib.IsTTY(os.Stderr) {
		go discoveryProgressLoop(set, walkDoneCh, numWorkers, walkWorkerUtilization)
	}
	<-walkDoneCh
	scanDuration := time.Since(startTime)

	pairPaths := set.PairPaths()
	totalCompared := len(pairPaths)

	var diffs []lib.DiffResult
	var pairJobs []lib.PairJob
	for _, relativePath := range pairPaths {
		pairJobs = append(pairJobs, lib.PairJob{Rel: relativePath})
	}
	progressCounts.TotalPairs = int32(len(pairJobs))
	if len(pairJobs) > 0 {
		progressCounts.WorkerProcessed = make([]int32, numWorkers)
	}

	compareWorkerUtilization := lib.NewWorkerUtilization(numWorkers, utilWindowTicks)

	compareStart := time.Now()
	pairCh := make(chan lib.PairJob, len(pairJobs)+1)
	go func() {
		for _, job := range pairJobs {
			pairCh <- job
		}
		close(pairCh)
	}()
	go lib.RunWorkers(left, right, numWorkers, hashAlg, hashThreshold, pairCh, compareResultCh, progressCounts, compareWorkerUtilization)

	compareDoneCh := make(chan struct{})
	if !quiet && lib.IsTTY(os.Stderr) {
		go progressLoop(progressCounts, compareDoneCh, numWorkers, compareWorkerUtilization)
	}
	for result := range compareResultCh {
		reportCompareResult(result, &diffs, logger)
	}
	close(compareDoneCh)
	compareDuration := time.Since(compareStart)
	differentCount := len(diffs)
	sameCount := totalCompared - differentCount
	if sameCount < 0 {
		sameCount = 0
	}
	leftOnlyPaths := set.LeftOnlyPaths()
	leftOnlyCount := 0
	for _, relativePath := range leftOnlyPaths {
		path := filepath.Join(left, relativePath)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			diffs = append(diffs, lib.DiffResult{Rel: relativePath, Reason: "left only", LeftSize: info.Size(), LeftMtime: info.ModTime().Truncate(time.Second), LeftOnly: true})
			leftOnlyCount++
		}
	}
	rightOnlyPaths := set.RightOnlyPaths()
	rightOnlyCount := 0
	for _, relativePath := range rightOnlyPaths {
		path := filepath.Join(right, relativePath)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			diffs = append(diffs, lib.DiffResult{Rel: relativePath, Reason: "right only", RightSize: info.Size(), RightMtime: info.ModTime().Truncate(time.Second)})
			rightOnlyCount++
		}
	}
	logger.Log("left-only files: " + fmt.Sprintf("%d", len(leftOnlyPaths)))
	for _, relativePath := range leftOnlyPaths {
		logger.Log("  " + relativePath)
	}
	logger.Log("right-only files: " + fmt.Sprintf("%d", len(rightOnlyPaths)))
	for _, relativePath := range rightOnlyPaths {
		logger.Log("  " + relativePath)
	}
	displaySummary(logger, !quiet, runSummary{
		leftDir:                  left,
		rightDir:                 right,
		totalCompared:            totalCompared,
		leftOnlyCount:            leftOnlyCount,
		rightOnlyCount:           rightOnlyCount,
		differentCount:           differentCount,
		sameCount:                sameCount,
		startTime:                startTime,
		scanDuration:             scanDuration,
		compareDuration:          compareDuration,
		workerUtilizationPercent: compareWorkerUtilization.UtilizedPercentWholeRun(),
	})
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

// reportCompareResult appends the result to diffs when different and logs either "different" or "identical" with reason to the logger.
func reportCompareResult(result lib.CompareResult, diffs *[]lib.DiffResult, logger *lib.Logger) {
	if result.Diff != nil {
		*diffs = append(*diffs, *result.Diff)
		logger.Log("different: " + result.RelativePath + " (" + result.Diff.Reason + ")")
	} else {
		logger.Log("identical: " + result.RelativePath + " (" + result.Reason + ")")
	}
}

// Prints "scanning: N left-only, N right-only, N pairs" to stderr on a ticker until doneCh closes. Appends the percentage of workers utilized (from workerUtilization.Tick()).
func discoveryProgressLoop(set *lib.DiscoveredSet, doneCh <-chan struct{}, numWorkers int, workerUtilization *lib.WorkerUtilization) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-doneCh:
			return
		case <-tick.C:
			leftOnly := set.LeftOnlyCount()
			rightOnly := set.RightOnlyCount()
			pairs := set.PairsCount()
			windowed := workerUtilization.Tick()
			total := workerUtilization.UtilizedPercentWholeRun()
			workStats := fmt.Sprintf(" [worker utilization 3s: %d%%, total: %d%%]", windowed, total)
			writeProgressLine("Scanning: %d left-only, %d right-only, %d pairs (%d workers)%s   ", leftOnly, rightOnly, pairs, numWorkers, workStats)
		}
	}
}

// Extrapolates remaining time from elapsed and (processed, pending) so we can show "~Xs remaining"; returns 0 if processed or pending is non-positive.
func estimateRemainingFromElapsed(elapsed time.Duration, processed, pending int32) time.Duration {
	if processed <= 0 || pending <= 0 {
		return 0
	}
	averagePerPair := elapsed / time.Duration(processed)
	return averagePerPair * time.Duration(pending)
}

// Uses progress counts and start time (from ProgressCounts) to compute remaining time; used by progressLoop with atomically loaded values.
func estimateRemainingDuration(processed, pending int32, startTimeUnixNano int64) time.Duration {
	if startTimeUnixNano == 0 {
		return 0
	}
	elapsed := time.Since(time.Unix(0, startTimeUnixNano))
	return estimateRemainingFromElapsed(elapsed, processed, pending)
}

// writeProgressLine overwrites the current stderr line with the formatted message, clearing to end of line first so no leftover text remains.
func writeProgressLine(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\r\033[K"+format, args...)
}

// Prints "comparing: N of M, ~Xs remaining" to stderr until doneCh closes. If workerUtilization is non-nil, appends the percentage of workers utilized in the last second (from workerUtilization.Tick()).
func progressLoop(progressCounts *lib.ProgressCounts, doneCh <-chan struct{}, numWorkers int, workerUtilization *lib.WorkerUtilization) {
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
			windowed := workerUtilization.Tick()
			total := workerUtilization.UtilizedPercentWholeRun()
			workStats := fmt.Sprintf(" [worker utilization 3s: %d%%, total: %d%%]", windowed, total)
			if totalPairs > 0 {
				pending := totalPairs - processedCount
				if pending < 0 {
					pending = 0
				}
				remaining := estimateRemainingDuration(processedCount, pending, startTimeNano)
				if remaining > 0 {
					writeProgressLine("Comparing: %d of %d, ~%s remaining (%d workers)%s   ", processedCount, totalPairs, remaining.Round(time.Second), numWorkers, workStats)
				} else {
					writeProgressLine("Comparing: %d of %d (%d workers)%s   ", processedCount, totalPairs, numWorkers, workStats)
				}
			} else {
				enqueuedCount := atomic.LoadInt32(&progressCounts.Enqueued)
				writeProgressLine("Processed %d, enqueued %d (%d workers)%s   ", processedCount, enqueuedCount, numWorkers, workStats)
			}
		}
	}
}
