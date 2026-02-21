//go:build !linux

package lib

func walkTreeWithBatch(root string, _ int, fn func(rel string, isDir bool)) {
	walkTreePortable(root, fn)
}
