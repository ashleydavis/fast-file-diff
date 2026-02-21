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

func (logger *Logger) TempDir() string { return logger.tempDir }

func (logger *Logger) Log(msg string) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if logger.mainFile != nil {
		fmt.Fprintln(logger.mainFile, msg)
		logger.mainFile.Sync()
	}
}

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

func (logger *Logger) NonFatalCount() int {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	return logger.nonFatal
}

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
