// Package imageutil provides small image-processing helpers used for avatars.
package imageutil

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	// Register decoders for the formats we accept on upload.
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
)

// ErrInvalidImage indicates the uploaded bytes could not be decoded as an image.
var ErrInvalidImage = errors.New("invalid image")

// ErrImageTooLarge indicates the image's pixel dimensions exceed the limit. It
// is reported from the header alone, before allocating the full pixel buffer.
var ErrImageTooLarge = errors.New("image dimensions too large")

// maxAvatarPixels caps the decoded surface (width*height) to defend against
// "decompression bomb" uploads: a tiny compressed file that would expand into a
// huge in-memory RGBA buffer (width*height*4 bytes). 8 MP ~= 32 MB decoded.
const maxAvatarPixels = 8 << 20

// ScaleAvatar decodes an arbitrary image and re-encodes it as a PNG that fits
// within maxSize on its longest side, preserving aspect ratio. Images already
// smaller than maxSize are re-encoded unchanged. The pixel dimensions are
// validated from the header before the full image is decoded.
func ScaleAvatar(r io.Reader, maxSize int) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading image: %w", err)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxAvatarPixels {
		return nil, fmt.Errorf("%w: %dx%d", ErrImageTooLarge, cfg.Width, cfg.Height)
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}

	dst := scaleToFit(src, maxSize)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, fmt.Errorf("encoding avatar: %w", err)
	}
	return buf.Bytes(), nil
}

// scaleToFit returns an image scaled so its longest side is at most maxSize,
// using bilinear interpolation. The original is returned when it already fits.
func scaleToFit(src image.Image, maxSize int) image.Image {
	b := src.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW == 0 || srcH == 0 {
		return src
	}
	if srcW <= maxSize && srcH <= maxSize {
		return src
	}

	scale := float64(maxSize) / float64(srcW)
	if srcH > srcW {
		scale = float64(maxSize) / float64(srcH)
	}
	dstW := int(float64(srcW) * scale)
	dstH := int(float64(srcH) * scale)
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	xRatio := float64(srcW) / float64(dstW)
	yRatio := float64(srcH) / float64(dstH)

	// Map each destination pixel center to the exact source coordinate (single
	// half-pixel correction; bilinearSample treats sx/sy as exact source coords).
	for y := 0; y < dstH; y++ {
		sy := float64(b.Min.Y) + (float64(y)+0.5)*yRatio - 0.5
		for x := 0; x < dstW; x++ {
			sx := float64(b.Min.X) + (float64(x)+0.5)*xRatio - 0.5
			r, g, bl, a := bilinearSample(src, b, sx, sy)
			dst.Set(x, y, rgba(r, g, bl, a))
		}
	}
	return dst
}
