package lib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// resolvePath resolves a relative path against root and returns the absolute path
// if it stays under root; otherwise returns an error (path traversal rejected).
func resolvePath(root, rel string) (string, error) {
	if rel == "" {
		return filepath.Clean(root), nil
	}
	abs := filepath.Join(root, rel)
	clean := filepath.Clean(abs)
	rootClean := filepath.Clean(root)
	if !pathUnder(clean, rootClean) {
		return "", errors.New("path escapes root")
	}
	return clean, nil
}

// pathUnder reports whether path is under or equal to root.
func pathUnder(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// PathPool interns relative path strings to avoid duplicates.
type PathPool struct {
	mu   sync.Mutex
	seen map[string]string
}

// NewPathPool returns a new path pool.
func NewPathPool() *PathPool {
	return &PathPool{seen: make(map[string]string)}
}

// Intern returns the same string for equal inputs, deduplicating storage.
func (p *PathPool) Intern(rel string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if cached, ok := p.seen[rel]; ok {
		return cached
	}
	p.seen[rel] = rel
	return rel
}

// EnsureDir returns nil if path is an existing directory; otherwise an error.
func EnsureDir(path string) error {
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
