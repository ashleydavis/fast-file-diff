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
	tempDir    string
	mainPath   string
	errorPath  string
	reqCh      chan logRequest
	errorCount int
	closeOnce  sync.Once
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
	logger := &Logger{tempDir: tmp, mainPath: mainPath, errorPath: errorPath, reqCh: make(chan logRequest)}
	go logger.run(mainFile, errorFile)
	return logger, nil
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
	for req := range logger.reqCh {
		switch req.kind {
		case reqLog:
			if mainFile != nil {
				fmt.Fprintln(&mainBuf, req.msg)
				if mainBuf.Len() >= mainLogFlushThreshold {
					flushMain()
				}
			}
			close(req.done)
		case reqLogError:
			logger.errorCount++
			flushMain() // flush before error so order is preserved
			if mainFile != nil {
				fmt.Fprintln(mainFile, "error:", req.err.Error())
				mainFile.Sync()
			}
			if errorFile != nil {
				fmt.Fprintln(errorFile, req.err.Error())
				errorFile.Sync()
			}
			close(req.done)
		case reqFatal:
			flushMain()
			msg := req.err.Error()
			if mainFile != nil {
				fmt.Fprintln(mainFile, "fatal:", msg)
				mainFile.Sync()
			}
			if errorFile != nil {
				fmt.Fprintln(errorFile, msg)
				errorFile.Sync()
			}
			close(req.done)
		case reqGetErrorCount:
			req.countResp <- logger.errorCount
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
			close(req.done)
			return
		}
	}
}

// Returns the temp directory path so callers can inspect log files or pass to other tools.
func (logger *Logger) TempDir() string { return logger.tempDir }

// Appends a line to the main log and syncs; used for normal progress/diff messages. Skips if already closed.
func (logger *Logger) Log(msg string) {
	done := make(chan struct{})
	logger.reqCh <- logRequest{kind: reqLog, msg: msg, done: done}
	<-done
}

// Writes the error to both main and error logs and increments the non-fatal count; used so we can report "N errors, check error log" at the end without exiting.
func (logger *Logger) LogError(err error) {
	done := make(chan struct{})
	logger.reqCh <- logRequest{kind: reqLogError, err: err, done: done}
	<-done
}

// Logs the error to both files, prints to stderr, then exits with FatalExitCode. Used for unrecoverable setup failures (e.g. can't create logger).
func (logger *Logger) Fatal(err error) {
	done := make(chan struct{})
	logger.reqCh <- logRequest{kind: reqFatal, err: err, done: done}
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
	logger.reqCh <- logRequest{kind: reqGetErrorCount, countResp: resp}
	return <-resp
}

// Closes both log files; the worker exits so later Log/LogError calls will block. Safe to call multiple times.
func (logger *Logger) Close() {
	logger.closeOnce.Do(func() {
		done := make(chan struct{})
		logger.reqCh <- logRequest{kind: reqClose, done: done}
		<-done
		close(logger.reqCh)
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
