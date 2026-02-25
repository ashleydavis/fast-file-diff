package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Exit code used when Fatal is called so callers can distinguish fatal errors from usage errors.
const FatalExitCode = 2

// requestKind identifies which operation the worker should perform.
type requestKind int

const (
	reqLog requestKind = iota
	reqLogError
	reqFatal
	reqClose
	reqGetErrorCount
	reqFlush
)

// logRequest is sent to the logger worker; only fields for the request kind are used.
type logRequest struct {
	kind      requestKind
	msg       string
	err       error
	done      chan struct{}
	countResp chan<- int
}

// Logger writes to a main log and a separate error log under a temp dir; used so we can report log paths and keep errors in one place. Safe for concurrent use via a single worker goroutine and channel.
type Logger struct {
	tempDir        string
	mainPath       string
	errorPath      string
	requestChannel chan logRequest
	errorCount     int
	closeOnce      sync.Once
}

// Log is the global logger, created at startup so any module can access it.
var Log = NewLogger()

// Creates a temp dir and two log files (main + errors) with dated names; callers should defer Close and optionally PrintLogPaths so users know where to look. Panics on failure.
func NewLogger() *Logger {
	tmp, err := os.MkdirTemp("", "ffd-*")
	if err != nil {
		panic("logger: " + err.Error())
	}
	date := time.Now().Format("20060102")
	base := filepath.Join(tmp, fmt.Sprintf("ffd-%s-001", date))
	mainPath := base + "-main.log"
	errorPath := base + "-errors.log"
	mainFile, err := os.Create(mainPath)
	if err != nil {
		os.RemoveAll(tmp)
		panic("logger: " + err.Error())
	}
	errorFile, err := os.Create(errorPath)
	if err != nil {
		mainFile.Close()
		os.RemoveAll(tmp)
		panic("logger: " + err.Error())
	}
	logger := &Logger{tempDir: tmp, mainPath: mainPath, errorPath: errorPath, requestChannel: make(chan logRequest)}
	go logger.run(mainFile, errorFile)
	return logger
}

// mainLogFlushThreshold is the buffer size in bytes at which we write and sync the main log to disk.
const mainLogFlushThreshold = 10 * 1024 * 1024 // 10 MiB

// run is the single worker that owns the log files and processes all log requests.
// Normal Log() calls are buffered; the buffer is written and synced every ~10 MiB or on Close.
func (logger *Logger) run(mainFile, errorFile *os.File) {
	var mainBuf bytes.Buffer
	flushMain := func() {
		if mainFile != nil && mainBuf.Len() > 0 {
			mainBuf.WriteTo(mainFile)
			mainFile.Sync()
			mainBuf.Reset()
		}
	}
	for request := range logger.requestChannel {
		switch request.kind {
		case reqLog:
			if mainFile != nil {
				fmt.Fprintln(&mainBuf, request.msg)
				if mainBuf.Len() >= mainLogFlushThreshold {
					flushMain()
				}
			}
			close(request.done)
		case reqLogError:
			logger.errorCount++
			flushMain() // flush before error so order is preserved
			if mainFile != nil {
				fmt.Fprintln(mainFile, "error:", request.err.Error())
				mainFile.Sync()
			}
			if errorFile != nil {
				fmt.Fprintln(errorFile, request.err.Error())
				errorFile.Sync()
			}
			close(request.done)
		case reqFatal:
			flushMain()
			msg := request.err.Error()
			if mainFile != nil {
				fmt.Fprintln(mainFile, "fatal:", msg)
				mainFile.Sync()
			}
			if errorFile != nil {
				fmt.Fprintln(errorFile, msg)
				errorFile.Sync()
			}
			close(request.done)
		case reqGetErrorCount:
			request.countResp <- logger.errorCount
		case reqFlush:
			flushMain()
			close(request.done)
		case reqClose:
			flushMain()
			if mainFile != nil {
				mainFile.Close()
				mainFile = nil
			}
			if errorFile != nil {
				errorFile.Close()
				errorFile = nil
			}
			close(request.done)
			return
		}
	}
}

// Returns the temp directory path so callers can inspect log files or pass to other tools.
func (logger *Logger) TempDir() string { return logger.tempDir }

// Write appends a line to the main log and syncs; used for normal progress/diff messages. Skips if already closed.
func (logger *Logger) Write(message string) {
	done := make(chan struct{})
	logger.requestChannel <- logRequest{kind: reqLog, msg: message, done: done}
	<-done
}

// Writes the error to both main and error logs and increments the non-fatal count; used so we can report "N errors, check error log" at the end without exiting.
func (logger *Logger) LogError(err error) {
	done := make(chan struct{})
	logger.requestChannel <- logRequest{kind: reqLogError, err: err, done: done}
	<-done
}

// Logs the error to both files, prints to stderr, then exits with FatalExitCode. Used for unrecoverable setup failures (e.g. can't create logger).
func (logger *Logger) Fatal(err error) {
	done := make(chan struct{})
	logger.requestChannel <- logRequest{kind: reqFatal, err: err, done: done}
	<-done
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(FatalExitCode)
}

// Prints main and error log paths to stderr so the user knows where to look; no-op when stdout isn't a TTY (e.g. in pipes) so we don't pollute script output.
func (logger *Logger) PrintLogPaths() {
	if !IsTTY(os.Stdout) {
		return
	}
	if logger.mainPath != "" {
		fmt.Fprintln(os.Stderr, "Main log:", logger.mainPath)
	}
	if logger.errorPath != "" {
		fmt.Fprintln(os.Stderr, "Error log:", logger.errorPath)
	}
}

// Returns how many times LogError was called; used to decide exit code and whether to tell the user to check the error log.
func (logger *Logger) ErrorCount() int {
	resp := make(chan int, 1)
	logger.requestChannel <- logRequest{kind: reqGetErrorCount, countResp: resp}
	return <-resp
}

// Flush writes and syncs the main log buffer to disk; use after compare so the log file is up to date before reporting left-only/right-only or exit.
func (logger *Logger) Flush() {
	done := make(chan struct{})
	logger.requestChannel <- logRequest{kind: reqFlush, done: done}
	<-done
}

// Closes both log files; the worker exits so later Log/LogError calls will block. Safe to call multiple times.
func (logger *Logger) Close() {
	logger.closeOnce.Do(func() {
		done := make(chan struct{})
		logger.requestChannel <- logRequest{kind: reqClose, done: done}
		<-done
		close(logger.requestChannel)
	})
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
