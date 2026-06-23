package services

import (
	"context"
	"fmt"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
)

func (s *ImportService) ImportFromZipUpload(ctx context.Context, zipBytes []byte, isAdmin bool) (*ImportResult, error) {
	if len(zipBytes) == 0 {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: "file is required"})
	}
	if err := validateZipBytes(zipBytes); err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: fmt.Sprintf("invalid zip: %v", err)})
	}
	return s.ImportFromZip(ctx, zipBytes, isAdmin)
}
