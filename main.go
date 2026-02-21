package main

import (
	"fmt"
	"os"
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
	defer logger.PrintLogPaths()
	logger.Log("started comparison")
	pool := newPathPool()
	set := newDiscoveredSet(pool)
	pairCh := make(chan string, pairQueueCap)
	resultCh := make(chan DiffResult, 256)
	progress := &progressCounts{}
	go walkBothTrees(left, right, dirBatchSize, logger, set, pairCh)
	runWorkers(left, right, numWorkers, hashAlg, hashThreshold, pairCh, resultCh, progress)
	doneCh := make(chan struct{})
	if isTTY(os.Stderr) {
		go progressLoop(progress, doneCh)
	}
	var diffs []DiffResult
	for r := range resultCh {
		diffs = append(diffs, r)
		logger.Log("diff: " + r.Rel + " " + r.Reason)
	}
	close(doneCh)
	switch outputFormat {
	case "table":
		formatTable(diffs, os.Stdout)
	case "json":
		formatJSON(diffs, os.Stdout)
	default:
		formatTextTree(diffs, os.Stdout)
	}
	return nil
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
			fmt.Fprintf(os.Stderr, "\rprocessed %d, pending %d   ", proc, pending)
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
