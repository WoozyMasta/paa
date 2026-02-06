package paa

import (
	"encoding/binary"
	"image"
	"image/color"
	"math"
)

// encodePixelFormat encodes img into raw uncompressed pixel data.
// Used for non-DXT formats (ARGB8, ARGB1555, ARGB4444, GRAYA/AI88).
func encodePixelFormat(format PaxType, img image.Image) ([]byte, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, ErrInvalidDimensions
	}

	switch format {
	case PaxARGB8:
		raw := make([]byte, w*h*4)
		i := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				raw[i+0] = c.B // store as BGRA
				raw[i+1] = c.G
				raw[i+2] = c.R
				raw[i+3] = c.A
				i += 4
			}
		}
		return raw, nil

	case PaxARGBA5:
		raw := make([]byte, w*h*2)
		i := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				a := uint16(0)
				if c.A >= 128 {
					a = 1
				}
				r := uint16(c.R >> 3)
				g := uint16(c.G >> 3)
				bl := uint16(c.B >> 3)
				p := (a << 15) | (r << 10) | (g << 5) | bl
				binary.LittleEndian.PutUint16(raw[i:], p)
				i += 2
			}
		}
		return raw, nil

	case PaxARGB4:
		raw := make([]byte, w*h*2)
		i := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				r := uint16(c.R >> 4)
				g := uint16(c.G >> 4)
				bl := uint16(c.B >> 4)
				a := uint16(c.A >> 4)
				p := (a << 12) | (bl << 8) | (g << 4) | r
				binary.LittleEndian.PutUint16(raw[i:], p)
				i += 2
			}
		}
		return raw, nil

	case PaxGRAYA:
		raw := make([]byte, w*h*2)
		i := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				l := uint8(math.Round(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)))
				raw[i+0] = l
				raw[i+1] = c.A
				i += 2
			}
		}
		return raw, nil

	default:
		return nil, ErrUnsupportedPixelFmt
	}
}
