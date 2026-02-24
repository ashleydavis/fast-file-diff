package lib

import (
	"io/fs"
	"os"
	"path/filepath"
)

// WalkTreeCollectFileInfo walks the tree at root using batched ReadDir, uses DirEntry.Info() for each file (no os.Stat), and returns one FileInfo per regular file. Mtime is normalized to 1-second granularity. Used by walk-left and walk-right phases.
func WalkTreeCollectFileInfo(root string, dirBatchSize int) []FileInfo {
	var collected []FileInfo
	walkCollect(root, "", dirBatchSize, &collected)
	return collected
}

// walkCollect recursively processes the directory root/relDir, appending FileInfo for each regular file to out. Skips symlinks and non-regular files; uses entry.Info() for size and mtime.
func walkCollect(root, relDir string, batchSize int, out *[]FileInfo) {
	absPath := filepath.Join(root, relDir)
	dirFile, err := os.Open(absPath)
	if err != nil {
		return
	}
	defer dirFile.Close()
	for {
		entries, err := dirFile.ReadDir(batchSize)
		if err != nil {
			return
		}
		if len(entries) == 0 {
			break
		}
		for _, entry := range entries {
			name := entry.Name()
			if name == "." || name == ".." {
				continue
			}
			relPath := name
			if relDir != "" {
				relPath = filepath.Join(relDir, name)
			}
			if entry.IsDir() {
				walkCollect(root, relPath, batchSize, out)
				continue
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				continue
			}
			if entry.Type()&fs.ModeType != 0 {
				continue
			}
			fileInfo, err := entry.Info()
			if err != nil {
				continue
			}
			*out = append(*out, FileInfo{
				Rel:   relPath,
				Size:  fileInfo.Size(),
				Mtime: NormalizeMtime(fileInfo.ModTime()),
				Hash:  "",
			})
		}
	}
}
