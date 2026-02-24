package lib

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cespare/xxhash/v2"
)

// HashFile hashes the file at path with the given algorithm; files smaller than threshold are read fully, larger ones are streamed. Exported for use by the phased pipeline (hash-left, hash-right).
func HashFile(path, algorithm string, threshold int) (string, error) {
	return hashFile(path, algorithm, threshold)
}

// hashFile is the internal implementation; HashFile is the exported wrapper.
func hashFile(path, algorithm string, threshold int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	size := info.Size()
	if size < 0 {
		size = 0
	}
	if int(size) < threshold {
		return hashFull(file, algorithm, int(size))
	}
	return hashStream(file, algorithm, threshold)
}

// Pool of 10MiB buffers for streaming hash; reused across hashStream calls to avoid allocating per file.
var bufPool = sync.Pool{
	New: func() interface{} {
		buffer := make([]byte, 10*1024*1024)
		return &buffer
	},
}

// Reads size bytes into a single buffer and hashes with hashBytes. Used for small files so we don't spin up the streaming path.
func hashFull(reader io.Reader, algorithm string, size int) (string, error) {
	fullBuffer := make([]byte, size)
	if _, err := io.ReadFull(reader, fullBuffer); err != nil {
		return "", err
	}
	return hashBytes(fullBuffer, algorithm)
}

// Hashes by reading in bufSize chunks and feeding the algorithm incrementally; uses bufPool so we don't allocate a big buffer per file. Used for files over the threshold.
func hashStream(reader io.Reader, algorithm string, bufSize int) (string, error) {
	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)
	if cap(*buf) < bufSize {
		*buf = make([]byte, bufSize)
	}
	readBuffer := (*buf)[:bufSize]
	switch algorithm {
	case "xxhash":
		hasher := xxhash.New()
		for {
			bytesRead, err := reader.Read(readBuffer)
			if bytesRead > 0 {
				hasher.Write(readBuffer[:bytesRead])
			}
			if err == io.EOF {
				return fmt.Sprintf("%016x", hasher.Sum64()), nil
			}
			if err != nil {
				return "", err
			}
		}
	case "sha256":
		hashDigest := sha256.New()
		for {
			bytesRead, err := reader.Read(readBuffer)
			if bytesRead > 0 {
				hashDigest.Write(readBuffer[:bytesRead])
			}
			if err == io.EOF {
				return hex.EncodeToString(hashDigest.Sum(nil)), nil
			}
			if err != nil {
				return "", err
			}
		}
	case "md5":
		hashDigest := md5.New()
		for {
			bytesRead, err := reader.Read(readBuffer)
			if bytesRead > 0 {
				hashDigest.Write(readBuffer[:bytesRead])
			}
			if err == io.EOF {
				return hex.EncodeToString(hashDigest.Sum(nil)), nil
			}
			if err != nil {
				return "", err
			}
		}
	default:
		return "", fmt.Errorf("unknown hash algorithm: %s", algorithm)
	}
}

// Hashes a byte slice with xxhash, sha256, or md5; returns hex string. Used by hashFull and for tests; streaming path uses the same algorithms via incremental writers.
func hashBytes(data []byte, algorithm string) (string, error) {
	switch algorithm {
	case "xxhash":
		xxSum64 := xxhash.Sum64(data)
		return fmt.Sprintf("%016x", xxSum64), nil
	case "sha256":
		shaDigest := sha256.Sum256(data)
		return hex.EncodeToString(shaDigest[:]), nil
	case "md5":
		md5Digest := md5.Sum(data)
		return hex.EncodeToString(md5Digest[:]), nil
	default:
		return "", fmt.Errorf("unknown hash algorithm: %s", algorithm)
	}
}
