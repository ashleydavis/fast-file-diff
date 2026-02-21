//go:build !linux

package main

// walkTreeWithBatch uses the portable implementation (batch size ignored on non-Linux).
func walkTreeWithBatch(root string, _ int, fn func(rel string, isDir bool)) {
	walkTreePortable(root, fn)
}
