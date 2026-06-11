package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolving storage path: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}
	return &LocalStorage{baseDir: abs}, nil
}

func (s *LocalStorage) Save(_ context.Context, key string, r io.Reader) (FileInfo, error) {
	if err := s.validateKey(key); err != nil {
		return FileInfo{}, err
	}

 fullPath := s.fullPath(key)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return FileInfo{}, fmt.Errorf("creating directories: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "*.tmp")
	if err != nil {
		return FileInfo{}, fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	hash := sha256.New()
	size, err := io.Copy(tmp, io.TeeReader(r, hash))
	if err != nil {
		tmp.Close()
		return FileInfo{}, fmt.Errorf("writing file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return FileInfo{}, fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, fullPath); err != nil {
		return FileInfo{}, fmt.Errorf("renaming temp file: %w", err)
	}

	return FileInfo{Size: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	if err := s.validateKey(key); err != nil {
		return nil, err
	}
	f, err := os.Open(s.fullPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, key)
		}
		return nil, fmt.Errorf("opening file: %w", err)
	}
	return f, nil
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	if err := s.validateKey(key); err != nil {
		return err
	}
	err := os.Remove(s.fullPath(key))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting file: %w", err)
	}
	return nil
}

func (s *LocalStorage) Stat(_ context.Context, key string) (FileInfo, error) {
	if err := s.validateKey(key); err != nil {
		return FileInfo{}, err
	}
 fullPath := s.fullPath(key)
	fi, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileInfo{}, fmt.Errorf("%w: %s", ErrFileNotFound, key)
		}
		return FileInfo{}, fmt.Errorf("stating file: %w", err)
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("opening file for hash: %w", err)
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return FileInfo{}, fmt.Errorf("computing hash: %w", err)
	}

	return FileInfo{Size: fi.Size(), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func (s *LocalStorage) fullPath(key string) string {
	return filepath.Join(s.baseDir, key)
}

func (s *LocalStorage) validateKey(key string) error {
	if strings.Contains(key, "..") {
		return fmt.Errorf("%w: key contains path traversal", ErrInvalidKey)
	}
	cleaned := filepath.Clean(filepath.Join(s.baseDir, key))
	if !strings.HasPrefix(cleaned, s.baseDir) {
		return fmt.Errorf("%w: key escapes storage directory", ErrInvalidKey)
	}
	return nil
}
