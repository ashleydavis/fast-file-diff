package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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

var rootCmd = &cobra.Command{
	Use:   "ffd <left-dir> <right-dir>",
	Short: "Fast file diff between two directory trees",
	Long:  "Compare two directory trees recursively. Left dir and right dir are required positional arguments.",
	Args:  cobra.MatchAll(cobra.ArbitraryArgs, requireZeroOrTwoArgs),
	RunE:  runRoot,
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
	walkTree(left, func(rel string, isDir bool) {
		if isDir {
			logger.Log("dir: " + rel)
		} else {
			logger.Log("file: " + rel)
		}
	})
	_ = right
	// No diff logic yet; single tree walk and log only.
	return nil
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
