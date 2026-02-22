//go:build !linux

package lib

// On non-Linux, batch size is ignored and we use walkTreePortable so behavior is consistent without Linux-specific Readdir batching.
func walkTreeWithBatch(root string, _ int, walkFileFunc WalkFileFunc) {
	walkTreePortable(root, walkFileFunc)
}
