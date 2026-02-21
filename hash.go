package main

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

func hashFile(path, algorithm string, threshold int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := info.Size()
	if size < 0 {
		size = 0
	}
	if int(size) < threshold {
		return hashFull(f, algorithm, int(size))
	}
	return hashStream(f, algorithm, threshold)
}

var bufPool = sync.Pool{
	New: func() interface{} {
		buffer := make([]byte, 10*1024*1024)
		return &buffer
	},
}

func hashFull(r io.Reader, algorithm string, size int) (string, error) {
	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return hashBytes(buf, algorithm)
}

func hashStream(r io.Reader, algorithm string, bufSize int) (string, error) {
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
			n, err := r.Read(readBuffer)
			if n > 0 {
				hasher.Write(readBuffer[:n])
			}
			if err == io.EOF {
				return fmt.Sprintf("%016x", hasher.Sum64()), nil
			}
			if err != nil {
				return "", err
			}
		}
	case "sha256":
		hash := sha256.New()
		for {
			n, err := r.Read(readBuffer)
			if n > 0 {
				hash.Write(readBuffer[:n])
			}
			if err == io.EOF {
				return hex.EncodeToString(hash.Sum(nil)), nil
			}
			if err != nil {
				return "", err
			}
		}
	case "md5":
		hash := md5.New()
		for {
			n, err := r.Read(readBuffer)
			if n > 0 {
				hash.Write(readBuffer[:n])
			}
			if err == io.EOF {
				return hex.EncodeToString(hash.Sum(nil)), nil
			}
			if err != nil {
				return "", err
			}
		}
	default:
		return "", fmt.Errorf("unknown hash algorithm: %s", algorithm)
	}
}

func hashBytes(data []byte, algorithm string) (string, error) {
	switch algorithm {
	case "xxhash":
		sum64 := xxhash.Sum64(data)
		return fmt.Sprintf("%016x", sum64), nil
	case "sha256":
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:]), nil
	case "md5":
		sum := md5.Sum(data)
		return hex.EncodeToString(sum[:]), nil
	default:
		return "", fmt.Errorf("unknown hash algorithm: %s", algorithm)
	}
}
