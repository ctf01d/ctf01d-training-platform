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

// ScaleAvatar decodes an arbitrary image and re-encodes it as a square-ish PNG
// that fits within maxSize on its longest side. The aspect ratio is preserved.
// Images already smaller than maxSize are re-encoded unchanged.
func ScaleAvatar(r io.Reader, maxSize int) ([]byte, error) {
	src, _, err := image.Decode(r)
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

	for y := 0; y < dstH; y++ {
		sy := (float64(y) + 0.5) * yRatio
		for x := 0; x < dstW; x++ {
			sx := (float64(x) + 0.5) * xRatio
			r, g, bl, a := bilinearSample(src, b, sx, sy)
			dst.Set(x, y, rgba(r, g, bl, a))
		}
	}
	return dst
}
