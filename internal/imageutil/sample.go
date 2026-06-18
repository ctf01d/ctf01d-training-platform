package imageutil

import (
	"image"
	"image/color"
	"math"
)

// bilinearSample returns the interpolated color at fractional coordinates
// (sx, sy) within src, clamped to the source bounds. Channels are returned as
// 8-bit values (0-255).
func bilinearSample(src image.Image, b image.Rectangle, sx, sy float64) (r, g, bl, a uint32) {
	x0 := int(math.Floor(sx - 0.5))
	y0 := int(math.Floor(sy - 0.5))
	dx := sx - 0.5 - float64(x0)
	dy := sy - 0.5 - float64(y0)

	r00, g00, b00, a00 := at(src, b, x0, y0)
	r10, g10, b10, a10 := at(src, b, x0+1, y0)
	r01, g01, b01, a01 := at(src, b, x0, y0+1)
	r11, g11, b11, a11 := at(src, b, x0+1, y0+1)

	r = lerp2(r00, r10, r01, r11, dx, dy)
	g = lerp2(g00, g10, g01, g11, dx, dy)
	bl = lerp2(b00, b10, b01, b11, dx, dy)
	a = lerp2(a00, a10, a01, a11, dx, dy)
	return
}

// at returns the 8-bit RGBA channels of the pixel at (x, y), clamped to bounds.
func at(src image.Image, b image.Rectangle, x, y int) (r, g, bl, a float64) {
	if x < b.Min.X {
		x = b.Min.X
	}
	if x >= b.Max.X {
		x = b.Max.X - 1
	}
	if y < b.Min.Y {
		y = b.Min.Y
	}
	if y >= b.Max.Y {
		y = b.Max.Y - 1
	}
	cr, cg, cb, ca := src.At(x, y).RGBA()
	return float64(cr >> 8), float64(cg >> 8), float64(cb >> 8), float64(ca >> 8)
}

func lerp2(c00, c10, c01, c11, dx, dy float64) uint32 {
	top := c00 + (c10-c00)*dx
	bottom := c01 + (c11-c01)*dx
	v := top + (bottom-top)*dy
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return uint32(v + 0.5)
}

func rgba(r, g, b, a uint32) color.RGBA {
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}
