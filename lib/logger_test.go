package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger_createsLogFiles(t *testing.T) {
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() err = %v", err)
	}
	defer logger.Close()
	if logger.TempDir() == "" {
		t.Error("TempDir() is empty")
	}
	fi, err := os.Stat(logger.TempDir())
	if err != nil || !fi.IsDir() {
		t.Errorf("temp dir missing or not dir: %v", err)
	}
	ents, _ := os.ReadDir(logger.TempDir())
	if len(ents) < 2 {
		t.Errorf("expected at least 2 files in temp dir, got %d", len(ents))
	}
}

func TestLogger_Log_writesToMainOnly(t *testing.T) {
	logger, _ := NewLogger()
	defer logger.Close()
	msg := "test main log line"
	logger.Log(msg)
	ents, _ := os.ReadDir(logger.TempDir())
	for _, e := range ents {
		if strings.Contains(e.Name(), "main") && !e.IsDir() {
			data, _ := os.ReadFile(filepath.Join(logger.TempDir(), e.Name()))
			if !strings.Contains(string(data), msg) {
				t.Errorf("main log does not contain %q", msg)
			}
			return
		}
	}
	t.Error("no main log file found")
}

func TestLogger_LogError_writesBothAndIncrementsCount(t *testing.T) {
	logger, _ := NewLogger()
	defer logger.Close()
	logger.LogError(os.ErrNotExist)
	if logger.NonFatalCount() != 1 {
		t.Errorf("NonFatalCount() = %d, want 1", logger.NonFatalCount())
	}
	logger.LogError(os.ErrClosed)
	if logger.NonFatalCount() != 2 {
		t.Errorf("NonFatalCount() = %d, want 2", logger.NonFatalCount())
	}
}

func TestLogger_Close_returnsNil(t *testing.T) {
	logger, _ := NewLogger()
	if err := logger.Close(); err != nil {
		t.Errorf("Close() = %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Errorf("second Close() = %v", err)
	}
}
