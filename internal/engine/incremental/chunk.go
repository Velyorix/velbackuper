package incremental

import (
	"encoding/hex"
	"io"

	"github.com/zeebo/blake3"
)

const (
	ChunkSizeMinMB = 4
	ChunkSizeMaxMB = 16
	ChunkSizeMin   = ChunkSizeMinMB * 1024 * 1024
	ChunkSizeMax   = ChunkSizeMaxMB * 1024 * 1024

	DefaultHashPrefixLen = 2
)

func ReadChunks(r io.Reader, chunkSize int64, fn func(chunk []byte) error) error {
	if chunkSize < ChunkSizeMin {
		chunkSize = ChunkSizeMin
	}
	if chunkSize > ChunkSizeMax {
		chunkSize = ChunkSizeMax
	}
	buf := make([]byte, chunkSize)
	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			if callErr := fn(buf[:n]); callErr != nil {
				return callErr
			}
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func HashChunkHex(chunk []byte) string {
	sum := blake3.Sum256(chunk)
	return hex.EncodeToString(sum[:])
}

func ObjectKeyPrefix(hash string, n int) string {
	if n <= 0 || len(hash) < n {
		return hash
	}
	return hash[:n]
}
