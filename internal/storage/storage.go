package storage

import (
	"context"
	"io"
)

type FileInfo struct {
	Size   int64
	SHA256 string
}

type Storage interface {
	Save(ctx context.Context, key string, r io.Reader) (FileInfo, error)
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
	Stat(ctx context.Context, key string) (FileInfo, error)
}
