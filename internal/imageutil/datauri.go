package imageutil

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// AvatarMaxDimension is the longest-side pixel limit applied to stored avatars
// and entity (team/service) icons. Larger images are scaled down to fit.
const AvatarMaxDimension = 256

// maxAvatarDataURIBytes caps the length of an inline data: URI we will decode.
// It bounds the base64 allocation before ScaleAvatar's pixel-budget check kicks
// in, so an oversized payload is rejected cheaply rather than buffered whole.
// ~12 MB of base64 decodes to ~9 MB, comfortably above any real 256px icon.
const maxAvatarDataURIBytes = 12 << 20

// ErrUnsupportedAvatar indicates an avatar reference that is neither a decodable
// inline image nor an http(s) URL (e.g. javascript:, ftp:, file:, or junk).
var ErrUnsupportedAvatar = errors.New("unsupported avatar reference")

const (
	dataURIScheme               = "data:"
	base64Marker                = ";base64,"
	pngDataURIPrefix            = "data:image/png;base64,"
	unsupportedAvatarPreviewLen = 16
)

// NormalizeAvatarURL canonicalises an avatar reference. Inline data: images are
// re-encoded to a PNG data URI with their alpha channel preserved, scaled to fit
// maxSize on the longest side, keeping team and service icons transparent
// regardless of the format they were uploaded in. http(s) URLs pass through
// unchanged, and nil/empty values are returned as-is. Any other scheme is
// rejected with ErrUnsupportedAvatar so a stray javascript:/ftp:/file: value
// cannot be stored (a stored-XSS guard). A data: URI whose payload is not a
// decodable image yields ErrInvalidImage (or ErrImageTooLarge).
func NormalizeAvatarURL(u *string, maxSize int) (*string, error) {
	if u == nil || *u == "" {
		return u, nil
	}
	if strings.HasPrefix(*u, dataURIScheme) {
		normalized, err := normalizeDataURI(*u, maxSize)
		if err != nil {
			return nil, err
		}
		return &normalized, nil
	}
	if !isHTTPURL(*u) {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedAvatar, schemeOf(*u))
	}
	return u, nil
}

func normalizeDataURI(uri string, maxSize int) (string, error) {
	if len(uri) > maxAvatarDataURIBytes {
		return "", fmt.Errorf("%w: data URI too large", ErrImageTooLarge)
	}
	i := strings.Index(uri, base64Marker)
	if i < 0 {
		return "", fmt.Errorf("%w: data URI must be base64-encoded", ErrInvalidImage)
	}
	raw, err := base64.StdEncoding.DecodeString(uri[i+len(base64Marker):])
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	scaled, err := ScaleAvatar(bytes.NewReader(raw), maxSize)
	if err != nil {
		return "", err
	}
	return pngDataURIPrefix + base64.StdEncoding.EncodeToString(scaled), nil
}

func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// schemeOf returns the URL scheme for diagnostics, or the (truncated) input when
// it cannot be parsed as one.
func schemeOf(s string) string {
	if u, err := url.Parse(s); err == nil && u.Scheme != "" {
		return u.Scheme
	}
	if len(s) > unsupportedAvatarPreviewLen {
		return s[:unsupportedAvatarPreviewLen]
	}
	return s
}
