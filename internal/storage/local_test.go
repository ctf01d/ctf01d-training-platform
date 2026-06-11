package storage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStorageSaveOpenDelete(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	ctx := context.Background()

	content := []byte("hello world")
	fi, err := ls.Save(ctx, "services/1/service.zip", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if fi.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", fi.Size, len(content))
	}
	if fi.SHA256 == "" {
		t.Error("SHA256 is empty")
	}

	rc, err := ls.Open(ctx, "services/1/service.zip")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rc); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("content mismatch: got %q, want %q", buf.String(), string(content))
	}

	statFi, err := ls.Stat(ctx, "services/1/service.zip")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if statFi.Size != fi.Size {
		t.Errorf("Stat Size = %d, want %d", statFi.Size, fi.Size)
	}
	if statFi.SHA256 != fi.SHA256 {
		t.Errorf("Stat SHA256 = %s, want %s", statFi.SHA256, fi.SHA256)
	}

	if err := ls.Delete(ctx, "services/1/service.zip"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = ls.Open(ctx, "services/1/service.zip")
	if !IsFileNotFound(err) {
		t.Errorf("Open after delete: got %v, want ErrFileNotFound", err)
	}
}

func TestLocalStoragePathTraversal(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	ctx := context.Background()

	cases := []string{
		"../etc/passwd",
		"../../secret",
		"foo/../../bar",
	}
	for _, key := range cases {
		_, err := ls.Save(ctx, key, strings.NewReader("x"))
		if err == nil {
			t.Errorf("Save(%q): expected error", key)
		}
		_, err = ls.Open(ctx, key)
		if err == nil {
			t.Errorf("Open(%q): expected error", key)
		}
		err = ls.Delete(ctx, key)
		if err == nil {
			t.Errorf("Delete(%q): expected error", key)
		}
		_, err = ls.Stat(ctx, key)
		if err == nil {
			t.Errorf("Stat(%q): expected error", key)
		}
	}
}

func TestLocalStorageSaveCreatesDirs(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	_, err = ls.Save(context.Background(), "a/b/c/d.txt", strings.NewReader("deep"))
	if err != nil {
		t.Fatalf("Save nested: %v", err)
	}

	expected := filepath.Join(dir, "a", "b", "c", "d.txt")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("file not created at expected path: %v", err)
	}
}

func TestLocalStorageDeleteNonexistent(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	err = ls.Delete(context.Background(), "nope.txt")
	if err != nil {
		t.Errorf("Delete nonexistent: got %v, want nil", err)
	}
}

func TestLocalStorageStatNonexistent(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	_, err = ls.Stat(context.Background(), "nope.txt")
	if !IsFileNotFound(err) {
		t.Errorf("Stat nonexistent: got %v, want ErrFileNotFound", err)
	}
}

func TestLocalStorageSHA256(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	content := []byte("test content for sha256")
	fi, err := ls.Save(context.Background(), "test.bin", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if len(fi.SHA256) != 64 {
		t.Errorf("SHA256 length = %d, want 64", len(fi.SHA256))
	}
}

func IsFileNotFound(err error) bool {
	return errors.Is(err, ErrFileNotFound)
}
