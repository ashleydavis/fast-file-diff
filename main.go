package main

import (
	"fmt"
	"os"
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
		fmt.Sprintf("  Left directory:         %s", s.leftDir),
		fmt.Sprintf("  Right directory:        %s", s.rightDir),
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
var phaseName string

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
	rootCmd.Flags().StringVar(&phaseName, "phase", "", "Run only this phase and print its duration to stderr (walk-left, walk-right, build-pairs, classify-pairs, hash-left, hash-right, compare-hashes); unset runs full diff")
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

// runLs discovers files under the given directory and prints each relative path to stdout (one per line) as they are found.
func runLs(cmd *cobra.Command, args []string) error {
	root := args[0]
	if err := lib.EnsureDir(root); err != nil {
		return fmt.Errorf("not a directory: %w", err)
	}
	start := time.Now()
	fileCh := make(chan lib.DiscoveredFile, 256)
	doneCh := make(chan struct{})
	var count atomic.Int32
	go func() {
		for file := range fileCh {
			fmt.Fprintln(cmd.OutOrStdout(), file.Rel)
			count.Add(1)
		}
		close(doneCh)
	}()
	util := lib.NewWorkerUtilization(numWorkers, 30)
	go lib.Discover([]lib.DirJob{{Root: root, RelDir: "", Side: lib.SideLeft}}, fileCh, dirBatchSize, numWorkers, util)
	<-doneCh
	elapsed := time.Since(start)
	fmt.Fprintf(cmd.ErrOrStderr(), "Listed %d files in %v\n", count.Load(), elapsed.Round(time.Millisecond))
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
	// When --phase is set, run only the phases needed for that phase, time only that phase, print duration to stderr, and exit.
	if phaseName != "" {
		if !lib.ValidPhase(phaseName) {
			fmt.Fprintf(os.Stderr, "invalid phase %q; valid: walk-left, walk-right, build-pairs, classify-pairs, hash-left, hash-right, compare-hashes\n", phaseName)
			return fmt.Errorf("invalid phase: %s", phaseName)
		}
		return runPhasedPhase(cmd, left, right)
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

	// Run the seven-phase pipeline (sequential, no workers).
	leftInfos := lib.PhaseWalkLeft(left, dirBatchSize)
	rightInfos := lib.PhaseWalkRight(right, dirBatchSize)
	buildResult := lib.PhaseBuildPairs(leftInfos, rightInfos)
	classifyResult := lib.PhaseClassifyPairs(buildResult.Pairs)
	lib.PhaseHashLeft(left, classifyResult.ContentCheckQueue, hashAlg, hashThreshold)
	lib.PhaseHashRight(right, classifyResult.ContentCheckQueue, hashAlg, hashThreshold)
	leftByPath := make(map[string]*lib.FileInfo)
	for i := range leftInfos {
		leftByPath[leftInfos[i].Rel] = &leftInfos[i]
	}
	rightByPath := make(map[string]*lib.FileInfo)
	for i := range rightInfos {
		rightByPath[rightInfos[i].Rel] = &rightInfos[i]
	}
	diffs := lib.PhaseCompareHashes(classifyResult.ContentCheckQueue, classifyResult.DifferingBySize, buildResult.LeftOnlyPaths, buildResult.RightOnlyPaths, leftByPath, rightByPath)

	totalCompared := len(buildResult.Pairs)
	leftOnlyCount := len(buildResult.LeftOnlyPaths)
	rightOnlyCount := len(buildResult.RightOnlyPaths)
	pairDiffsCount := len(diffs) - leftOnlyCount - rightOnlyCount
	differentCount := pairDiffsCount
	sameCount := totalCompared - differentCount
	if sameCount < 0 {
		sameCount = 0
	}
	elapsed := time.Since(startTime)
	logger.Log("left-only files: " + fmt.Sprintf("%d", len(buildResult.LeftOnlyPaths)))
	for _, relativePath := range buildResult.LeftOnlyPaths {
		logger.Log("  " + relativePath)
	}
	logger.Log("right-only files: " + fmt.Sprintf("%d", len(buildResult.RightOnlyPaths)))
	for _, relativePath := range buildResult.RightOnlyPaths {
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
		scanDuration:             elapsed,
		compareDuration:          elapsed,
		workerUtilizationPercent: 0,
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
	if logger.ErrorCount() > 0 {
		if !quiet {
			fmt.Fprintln(os.Stderr, "Errors occurred; check the error log for details.")
		}
		os.Exit(ExitNonFatal)
	}
	return nil
}

// runPhasedPhase runs only the phases needed for phaseName's inputs, then runs the requested phase, times only that phase, prints duration to stderr, and returns.
func runPhasedPhase(cmd *cobra.Command, left, right string) error {
	var leftInfos []lib.FileInfo
	var rightInfos []lib.FileInfo
	var buildResult lib.BuildPairsResult
	var classifyResult lib.ClassifyPairsResult

	runWalkLeft := func() { leftInfos = lib.PhaseWalkLeft(left, dirBatchSize) }
	runWalkRight := func() { rightInfos = lib.PhaseWalkRight(right, dirBatchSize) }
	runBuildPairs := func() { buildResult = lib.PhaseBuildPairs(leftInfos, rightInfos) }
	runClassifyPairs := func() { classifyResult = lib.PhaseClassifyPairs(buildResult.Pairs) }

	switch phaseName {
	case "walk-left":
		start := time.Now()
		runWalkLeft()
		fmt.Fprintf(cmd.ErrOrStderr(), "walk-left: %v\n", time.Since(start))
		return nil
	case "walk-right":
		start := time.Now()
		runWalkRight()
		fmt.Fprintf(cmd.ErrOrStderr(), "walk-right: %v\n", time.Since(start))
		return nil
	case "build-pairs":
		runWalkLeft()
		runWalkRight()
		start := time.Now()
		runBuildPairs()
		fmt.Fprintf(cmd.ErrOrStderr(), "build-pairs: %v\n", time.Since(start))
		return nil
	case "classify-pairs":
		runWalkLeft()
		runWalkRight()
		runBuildPairs()
		start := time.Now()
		runClassifyPairs()
		fmt.Fprintf(cmd.ErrOrStderr(), "classify-pairs: %v\n", time.Since(start))
		return nil
	case "hash-left":
		runWalkLeft()
		runWalkRight()
		runBuildPairs()
		runClassifyPairs()
		start := time.Now()
		lib.PhaseHashLeft(left, classifyResult.ContentCheckQueue, hashAlg, hashThreshold)
		fmt.Fprintf(cmd.ErrOrStderr(), "hash-left: %v\n", time.Since(start))
		return nil
	case "hash-right":
		runWalkLeft()
		runWalkRight()
		runBuildPairs()
		runClassifyPairs()
		start := time.Now()
		lib.PhaseHashRight(right, classifyResult.ContentCheckQueue, hashAlg, hashThreshold)
		fmt.Fprintf(cmd.ErrOrStderr(), "hash-right: %v\n", time.Since(start))
		return nil
	case "compare-hashes":
		runWalkLeft()
		runWalkRight()
		runBuildPairs()
		runClassifyPairs()
		lib.PhaseHashLeft(left, classifyResult.ContentCheckQueue, hashAlg, hashThreshold)
		lib.PhaseHashRight(right, classifyResult.ContentCheckQueue, hashAlg, hashThreshold)
		leftByPath := make(map[string]*lib.FileInfo)
		for i := range leftInfos {
			leftByPath[leftInfos[i].Rel] = &leftInfos[i]
		}
		rightByPath := make(map[string]*lib.FileInfo)
		for i := range rightInfos {
			rightByPath[rightInfos[i].Rel] = &rightInfos[i]
		}
		start := time.Now()
		_ = lib.PhaseCompareHashes(classifyResult.ContentCheckQueue, classifyResult.DifferingBySize, buildResult.LeftOnlyPaths, buildResult.RightOnlyPaths, leftByPath, rightByPath)
		fmt.Fprintf(cmd.ErrOrStderr(), "compare-hashes: %v\n", time.Since(start))
		return nil
	}
	return fmt.Errorf("unknown phase: %s", phaseName)
}
