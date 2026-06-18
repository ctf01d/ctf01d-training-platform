package imageutil

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func decodeDims(t *testing.T, data []byte) (int, int) {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	b := img.Bounds()
	return b.Dx(), b.Dy()
}

func TestScaleAvatar_RejectsOversizeDimensions(t *testing.T) {
	// 3000x3000 = 9 MP > maxAvatarPixels (8 MP). Must be rejected from the
	// header, before the full RGBA buffer is allocated.
	data := encodePNG(t, 3000, 3000)
	_, err := ScaleAvatar(bytes.NewReader(data), 256)
	if !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("expected ErrImageTooLarge, got %v", err)
	}
}

func TestScaleAvatar_InvalidImage(t *testing.T) {
	_, err := ScaleAvatar(bytes.NewReader([]byte("not an image")), 256)
	if !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected ErrInvalidImage, got %v", err)
	}
}

func TestScaleAvatar_ScalesDownPreservingAspect(t *testing.T) {
	data := encodePNG(t, 512, 256)
	out, err := ScaleAvatar(bytes.NewReader(data), 256)
	if err != nil {
		t.Fatalf("ScaleAvatar: %v", err)
	}
	w, h := decodeDims(t, out)
	if w != 256 || h != 128 {
		t.Errorf("got %dx%d, want 256x128", w, h)
	}
}

func TestScaleAvatar_SmallImageUnchanged(t *testing.T) {
	data := encodePNG(t, 100, 80)
	out, err := ScaleAvatar(bytes.NewReader(data), 256)
	if err != nil {
		t.Fatalf("ScaleAvatar: %v", err)
	}
	w, h := decodeDims(t, out)
	if w != 100 || h != 80 {
		t.Errorf("got %dx%d, want 100x80", w, h)
	}
}
