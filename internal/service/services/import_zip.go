package services

import (
	"context"
	"errors"
	"fmt"
)

func (s *ImportService) ImportFromZipUpload(ctx context.Context, zipBytes []byte, isAdmin bool) (*ImportResult, error) {
	if len(zipBytes) == 0 {
		return nil, errors.New("empty zip upload")
	}
	if err := validateZipBytes(zipBytes); err != nil {
		return nil, fmt.Errorf("invalid zip: %w", err)
	}
	return s.ImportFromZip(ctx, zipBytes, isAdmin)
}
