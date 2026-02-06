package texconfig

import (
	"image"
	"image/color"
)

// ApplyChannelSwizzle returns a new NRGBA image with the swizzle applied.
// It converts the input to NRGBA before channel mapping.
func ApplyChannelSwizzle(img image.Image, swz ChannelSwizzle) *image.NRGBA {
	b := img.Bounds()
	out := image.NewNRGBA(b)

	// Iterate over image
	for y := b.Min.Y; y < b.Max.Y; y++ {
		// Iterate over row
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			r, g, b2, a := swz.apply(c.R, c.G, c.B, c.A)

			out.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b2, A: a})
		}
	}

	return out
}
