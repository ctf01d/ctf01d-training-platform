package storage

import (
	"errors"
)

var (
	ErrFileNotFound = errors.New("file not found")
	ErrInvalidKey   = errors.New("invalid storage key")
)
