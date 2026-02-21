//go:build !linux

package lib

func walkTreeWithBatch(root string, _ int, fn WalkFileFunc) {
	walkTreePortable(root, fn)
}
