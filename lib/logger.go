package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Exit code used when Fatal is called so callers can distinguish fatal errors from usage errors.
const FatalExitCode = 2

// Writes to a main log and a separate error log under a temp dir; used so we can report log paths and keep errors in one place. Safe for concurrent use via mutex.
type Logger struct {
	tempDir   string
	mainPath  string
	errorPath string
	mainFile  *os.File
	errorFile *os.File
	nonFatal  int
	mu        sync.Mutex
}

// Creates a temp dir and two log files (main + errors) with dated names; callers should defer Close and optionally PrintLogPaths so users know where to look.
func NewLogger() (*Logger, error) {
	tmp, err := os.MkdirTemp("", "ffd-*")
	if err != nil {
		return nil, err
	}
	date := time.Now().Format("20060102")
	base := filepath.Join(tmp, fmt.Sprintf("ffd-%s-001", date))
	mainPath := base + "-main.log"
	errorPath := base + "-errors.log"
	mainFile, err := os.Create(mainPath)
	if err != nil {
		os.RemoveAll(tmp)
		return nil, err
	}
	errorFile, err := os.Create(errorPath)
	if err != nil {
		mainFile.Close()
		os.RemoveAll(tmp)
		return nil, err
	}
	return &Logger{tempDir: tmp, mainPath: mainPath, errorPath: errorPath, mainFile: mainFile, errorFile: errorFile}, nil
}

// Returns the temp directory path so callers can inspect log files or pass to other tools.
func (logger *Logger) TempDir() string { return logger.tempDir }

// Appends a line to the main log and syncs; used for normal progress/diff messages. Skips if already closed.
func (logger *Logger) Log(msg string) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if logger.mainFile != nil {
		fmt.Fprintln(logger.mainFile, msg)
		logger.mainFile.Sync()
	}
}

// Writes the error to both main and error logs and increments the non-fatal count; used so we can report "N errors, check error log" at the end without exiting.
func (logger *Logger) LogError(err error) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	logger.nonFatal++
	if logger.mainFile != nil {
		fmt.Fprintln(logger.mainFile, "error:", err.Error())
		logger.mainFile.Sync()
	}
	if logger.errorFile != nil {
		fmt.Fprintln(logger.errorFile, err.Error())
		logger.errorFile.Sync()
	}
}

// Logs the error to both files, prints to stderr, then exits with FatalExitCode. Used for unrecoverable setup failures (e.g. can't create logger).
func (logger *Logger) Fatal(err error) {
	logger.mu.Lock()
	msg := err.Error()
	if logger.mainFile != nil {
		fmt.Fprintln(logger.mainFile, "fatal:", msg)
		logger.mainFile.Sync()
	}
	if logger.errorFile != nil {
		fmt.Fprintln(logger.errorFile, msg)
		logger.errorFile.Sync()
	}
	logger.mu.Unlock()
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(FatalExitCode)
}

// Prints main and error log paths to stderr so the user knows where to look; no-op when stdout isn't a TTY (e.g. in pipes) so we don't pollute script output.
func (logger *Logger) PrintLogPaths() {
	if !IsTTY(os.Stdout) {
		return
	}
	logger.mu.Lock()
	mainPath := logger.mainPath
	errorPath := logger.errorPath
	logger.mu.Unlock()
	if mainPath != "" {
		fmt.Fprintln(os.Stderr, "Main log:", mainPath)
	}
	if errorPath != "" {
		fmt.Fprintln(os.Stderr, "Error log:", errorPath)
	}
}

// Returns how many times LogError was called; used to decide exit code and whether to tell the user to check the error log.
func (logger *Logger) NonFatalCount() int {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	return logger.nonFatal
}

// Closes both log files and clears references so later Log/LogError calls no-op. Returns the first close error if any.
func (logger *Logger) Close() error {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	var closeError error
	if logger.mainFile != nil {
		if closeErr := logger.mainFile.Close(); closeErr != nil && closeError == nil {
			closeError = closeErr
		}
		logger.mainFile = nil
	}
	if logger.errorFile != nil {
		if closeErr := logger.errorFile.Close(); closeErr != nil && closeError == nil {
			closeError = closeErr
		}
		logger.errorFile = nil
	}
	return closeError
}

// Reports whether file is a character device (terminal); used to decide whether to show progress and log paths so we don't spam non-interactive output.
func IsTTY(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
