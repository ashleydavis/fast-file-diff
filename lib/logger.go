package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const FatalExitCode = 2

type Logger struct {
	tempDir   string
	mainPath  string
	errorPath string
	mainFile  *os.File
	errorFile *os.File
	nonFatal  int
	mu        sync.Mutex
}

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

func (l *Logger) TempDir() string { return l.tempDir }

func (l *Logger) Log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.mainFile != nil {
		fmt.Fprintln(l.mainFile, msg)
		l.mainFile.Sync()
	}
}

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
	os.Exit(FatalExitCode)
}

func (l *Logger) PrintLogPaths() {
	if !IsTTY(os.Stdout) {
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

func (l *Logger) NonFatalCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.nonFatal
}

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

func IsTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
