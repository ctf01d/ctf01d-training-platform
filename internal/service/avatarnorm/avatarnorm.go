// Package avatarnorm normalizes inline avatar/icon images to transparent PNG
// data URIs, mapping decode failures to validation errors. It is shared by the
// team, service, and university services so their icons are stored in a single
// alpha-preserving format.
package avatarnorm

import (
	"errors"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/imageutil"
)

// Normalize re-encodes an inline data: image avatar to a transparent PNG data
// URI scaled to the standard avatar size, leaving nil, empty, and http(s) URL
// values untouched. Any other scheme, or a data URI that does not hold a
// decodable image, is reported as a validation error on the named field.
func Normalize(u *string, field string) (*string, error) {
	normalized, err := imageutil.NormalizeAvatarURL(u, imageutil.AvatarMaxDimension)
	if err != nil {
		switch {
		case errors.Is(err, imageutil.ErrUnsupportedAvatar):
			return nil, errs.NewValidationError(map[string]string{field: "must be an http(s) URL or data:image URI"})
		case errors.Is(err, imageutil.ErrInvalidImage), errors.Is(err, imageutil.ErrImageTooLarge):
			return nil, errs.NewValidationError(map[string]string{field: "must be a valid image"})
		default:
			return nil, err
		}
	}
	return normalized, nil
}
