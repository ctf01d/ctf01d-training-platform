package imageutil

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

// transparentPNGDataURI builds a data: URI for a w*h image whose left half is
// fully transparent and right half opaque, in the given image/* media type.
func transparentPNGDataURI(t *testing.T, w, h int, mediaType string) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := uint8(0)
			if x >= w/2 {
				a = 255
			}
			img.Set(x, y, color.NRGBA{R: 10, G: 200, B: 90, A: a})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestNormalizeAvatarURL_PreservesAlphaAndReencodesPNG(t *testing.T) {
	// Larger than AvatarMaxDimension so it also exercises the scaling path.
	in := transparentPNGDataURI(t, 512, 512, "image/png")
	out, err := NormalizeAvatarURL(&in, AvatarMaxDimension)
	if err != nil {
		t.Fatalf("NormalizeAvatarURL: %v", err)
	}
	if out == nil || !strings.HasPrefix(*out, pngDataURIPrefix) {
		t.Fatalf("expected png data URI, got %q", derefShort(out))
	}

	raw, err := base64.StdEncoding.DecodeString((*out)[len(pngDataURIPrefix):])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	b := img.Bounds()
	if b.Dx() > AvatarMaxDimension || b.Dy() > AvatarMaxDimension {
		t.Errorf("not scaled down: %dx%d", b.Dx(), b.Dy())
	}
	if _, _, _, a := img.At(b.Min.X+2, b.Min.Y+2).RGBA(); a>>8 != 0 {
		t.Errorf("transparent corner lost alpha: got %d, want 0", a>>8)
	}
	if _, _, _, a := img.At(b.Max.X-3, b.Min.Y+2).RGBA(); a>>8 != 255 {
		t.Errorf("opaque corner alpha changed: got %d, want 255", a>>8)
	}
}

func TestNormalizeAvatarURL_ConvertsNonPNGToPNG(t *testing.T) {
	// A data URI labelled image/jpeg but carrying PNG bytes must still come out
	// as a PNG data URI (we re-encode by decoded content, not the label).
	in := transparentPNGDataURI(t, 64, 64, "image/jpeg")
	out, err := NormalizeAvatarURL(&in, AvatarMaxDimension)
	if err != nil {
		t.Fatalf("NormalizeAvatarURL: %v", err)
	}
	if out == nil || !strings.HasPrefix(*out, pngDataURIPrefix) {
		t.Fatalf("expected png data URI, got %q", derefShort(out))
	}
}

func TestNormalizeAvatarURL_LeavesHTTPAndEmptyUnchanged(t *testing.T) {
	link := "https://example.com/logo.png"
	out, err := NormalizeAvatarURL(&link, AvatarMaxDimension)
	if err != nil || out == nil || *out != link {
		t.Fatalf("http URL should pass through unchanged: got %q, err %v", derefShort(out), err)
	}

	if out, err := NormalizeAvatarURL(nil, AvatarMaxDimension); err != nil || out != nil {
		t.Fatalf("nil should pass through: got %q, err %v", derefShort(out), err)
	}
}

func TestNormalizeAvatarURL_RejectsNonHTTPScheme(t *testing.T) {
	for _, bad := range []string{
		"javascript:alert(1)",
		"ftp://example.com/logo.png",
		"file:///etc/passwd",
		"not even a url",
	} {
		if _, err := NormalizeAvatarURL(&bad, AvatarMaxDimension); !errors.Is(err, ErrUnsupportedAvatar) {
			t.Errorf("%q: expected ErrUnsupportedAvatar, got %v", bad, err)
		}
	}
}

func TestNormalizeAvatarURL_RejectsOversizeDataURI(t *testing.T) {
	huge := "data:image/png;base64," + strings.Repeat("A", maxAvatarDataURIBytes)
	if _, err := NormalizeAvatarURL(&huge, AvatarMaxDimension); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("expected ErrImageTooLarge for oversize data URI, got %v", err)
	}
}

func TestNormalizeAvatarURL_RejectsGarbageDataURI(t *testing.T) {
	bad := "data:image/png;base64,not-valid-base64-@@@"
	if _, err := NormalizeAvatarURL(&bad, AvatarMaxDimension); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected ErrInvalidImage, got %v", err)
	}

	noMarker := "data:image/png,rawdata"
	if _, err := NormalizeAvatarURL(&noMarker, AvatarMaxDimension); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected ErrInvalidImage for non-base64 data URI, got %v", err)
	}
}

func derefShort(s *string) string {
	if s == nil {
		return "<nil>"
	}
	if len(*s) > 40 {
		return (*s)[:40] + "..."
	}
	return *s
}
