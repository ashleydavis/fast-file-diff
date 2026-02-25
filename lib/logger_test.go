package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger_createsLogFiles(t *testing.T) {
	logger := NewLogger()
	defer logger.Close()
	if logger.TempDir() == "" {
		t.Error("TempDir() is empty")
	}
	fileInfo, err := os.Stat(logger.TempDir())
	if err != nil || !fileInfo.IsDir() {
		t.Errorf("temp dir missing or not dir: %v", err)
	}
	entries, _ := os.ReadDir(logger.TempDir())
	if len(entries) < 2 {
		t.Errorf("expected at least 2 files in temp dir, got %d", len(entries))
	}
}

func TestLogger_Log_writesToMainOnly(t *testing.T) {
	logger := NewLogger()
	defer logger.Close()
	msg := "test main log line"
	logger.Write(msg)
	logger.Close() // flush buffer before reading file
	entries, _ := os.ReadDir(logger.TempDir())
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "main") && !entry.IsDir() {
			data, _ := os.ReadFile(filepath.Join(logger.TempDir(), entry.Name()))
			if !strings.Contains(string(data), msg) {
				t.Errorf("main log does not contain %q", msg)
			}
			return
		}
	}
	t.Error("no main log file found")
}

func TestLogger_LogError_writesBothAndIncrementsCount(t *testing.T) {
	logger := NewLogger()
	defer logger.Close()
	logger.LogError(os.ErrNotExist)
	if logger.ErrorCount() != 1 {
		t.Errorf("ErrorCount() = %d, want 1", logger.ErrorCount())
	}
	logger.LogError(os.ErrClosed)
	if logger.ErrorCount() != 2 {
		t.Errorf("ErrorCount() = %d, want 2", logger.ErrorCount())
	}
}

// TestLogger_Flush_writesBufferedToDisk verifies that Flush writes the main log buffer to disk without closing the logger.
func TestLogger_Flush_writesBufferedToDisk(t *testing.T) {
	logger := NewLogger()
	defer logger.Close()
	msg := "flush test line"
	logger.Write(msg)
	logger.Flush()
	entries, err := os.ReadDir(logger.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "main") && !entry.IsDir() {
			data, err := os.ReadFile(filepath.Join(logger.TempDir(), entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), msg) {
				t.Errorf("main log after Flush() does not contain %q", msg)
			}
			return
		}
	}
	t.Error("no main log file found")
}

func TestLogger_Close_idempotent(t *testing.T) {
	logger := NewLogger()
	logger.Close()
	logger.Close()
}

func TestIsTTY_regularFileReturnsFalse(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if IsTTY(f) {
		t.Error("IsTTY(regular file) should be false")
	}
}

func TestIsTTY_nilReturnsFalse(t *testing.T) {
	if IsTTY(nil) {
		t.Error("IsTTY(nil) should be false")
	}
}

func TestLogger_PrintLogPaths_doesNotPanic(t *testing.T) {
	logger := NewLogger()
	defer logger.Close()
	logger.PrintLogPaths()
}
