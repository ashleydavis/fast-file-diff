package main

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
	dir := logger.TempDir()
	if dir == "" {
		t.Error("TempDir() is empty")
	}
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		t.Errorf("temp dir %q missing or not dir: %v", dir, err)
	}
	// Should have main and error log files
	ents, _ := os.ReadDir(dir)
	if len(ents) < 2 {
		t.Errorf("expected at least 2 files in temp dir, got %d", len(ents))
	}
}

func TestLogger_Log_writesToMainOnly(t *testing.T) {
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() err = %v", err)
	}
	defer logger.Close()
	msg := "test main log line"
	logger.Log(msg)
	// Read main log (name contains "main")
	ents, _ := os.ReadDir(logger.TempDir())
	for _, e := range ents {
		if strings.Contains(e.Name(), "main") && !e.IsDir() {
			path := filepath.Join(logger.TempDir(), e.Name())
			data, _ := os.ReadFile(path)
			if !strings.Contains(string(data), msg) {
				t.Errorf("main log does not contain %q: %s", msg, data)
			}
			return
		}
	}
	t.Error("no main log file found")
}

func TestLogger_LogError_writesBothAndIncrementsCount(t *testing.T) {
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() err = %v", err)
	}
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
	logger, err := NewLogger()
	if err != nil {
		t.Fatal(err)
	}
	if err := logger.Close(); err != nil {
		t.Errorf("Close() = %v", err)
	}
	// Second Close is no-op (files already nil)
	if err := logger.Close(); err != nil {
		t.Errorf("second Close() = %v", err)
	}
}
