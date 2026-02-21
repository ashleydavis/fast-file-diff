package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger holds the logging state for one run: temp dir, main log, error log, non-fatal count.
// Safe for concurrent use.
type Logger struct {
	tempDir   string
	mainPath  string
	errorPath string
	mainFile  *os.File
	errorFile *os.File
	nonFatal  int
	mu        sync.Mutex
}

// NewLogger creates a temp dir and opens main and error log files with names
// ffd-YYYYMMDD-NNN-main.log and ffd-YYYYMMDD-NNN-errors.log.
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
	return &Logger{
		tempDir:   tmp,
		mainPath:  mainPath,
		errorPath: errorPath,
		mainFile:  mainFile,
		errorFile: errorFile,
	}, nil
}

// TempDir returns the logger's temp directory path.
func (l *Logger) TempDir() string { return l.tempDir }

// Log writes msg to the main log only.
func (l *Logger) Log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.mainFile != nil {
		fmt.Fprintln(l.mainFile, msg)
		l.mainFile.Sync()
	}
}

// LogError writes err to both logs and increments the non-fatal count.
func (l *Logger) LogError(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.nonFatal++
	if l.mainFile != nil {
		fmt.Fprintln(l.mainFile, "error:", err.Error())
		l.mainFile.Sync()
	}
	if l.errorFile != nil {
		fmt.Fprintln(l.errorFile, err.Error())
		l.errorFile.Sync()
	}
}

// Fatal writes err to both logs, prints to stderr, and exits with code 2.
func (l *Logger) Fatal(err error) {
	l.mu.Lock()
	msg := err.Error()
	if l.mainFile != nil {
		fmt.Fprintln(l.mainFile, "fatal:", msg)
		l.mainFile.Sync()
	}
	if l.errorFile != nil {
		fmt.Fprintln(l.errorFile, msg)
		l.errorFile.Sync()
	}
	l.mu.Unlock()
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(ExitFatal)
}

// PrintLogPaths prints the two log file paths to stderr. Skip when stdout is not a TTY.
func (l *Logger) PrintLogPaths() {
	if !isTTY(os.Stdout) {
		return
	}
	l.mu.Lock()
	mainPath := l.mainPath
	errorPath := l.errorPath
	l.mu.Unlock()
	if mainPath != "" {
		fmt.Fprintln(os.Stderr, "Main log:", mainPath)
	}
	if errorPath != "" {
		fmt.Fprintln(os.Stderr, "Error log:", errorPath)
	}
}

// NonFatalCount returns the current non-fatal error count.
func (l *Logger) NonFatalCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.nonFatal
}

// Close flushes and closes the log files.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var err error
	if l.mainFile != nil {
		if closeErr := l.mainFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		l.mainFile = nil
	}
	if l.errorFile != nil {
		if closeErr := l.errorFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		l.errorFile = nil
	}
	return err
}

// isTTY reports whether the file is a terminal (for progress/log path display).
func isTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
