package paa

import (
	"image"
	"image/color"
	"math"
)

// swizzleDXT5NM is the SWIZTAGG payload for DXT5 normal maps (_nohq/_nofhq).
var swizzleDXT5NM = [4]byte{0x05, 0x04, 0x02, 0x03}

// swizzleNormalMap (Encode): Tangent Space RGB (R=X, G=Y, B=Z) -> PAA nohq storage for SWIZTAGG.
// BIS wiki: *_nohq.paa swizzle = (Alpha negated in R, Red negated in A, G as-is, B as-is).
// Engine: R_display=255-A_stored, G_display=G_stored, B_display=B_stored, A_display=255-R_stored.
// We want (X,Y,Z,255) => store R=0, G=Y, B=Z, A=255-X.
func swizzleNormalMap(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8 := uint8(r >> 8) //nolint:gosec // r is 0..65535
			g8 := uint8(g >> 8) //nolint:gosec // g is 0..65535
			b8 := uint8(b >> 8) //nolint:gosec // b is 0..65535

			nx := (float64(r8)/255.0)*2.0 - 1.0
			ny := (float64(g8)/255.0)*2.0 - 1.0

			d := 1.0 - nx*nx - ny*ny
			if d > 0 {
				nzRec := math.Sqrt(d)
				z8Rec := uint8(clamp01(nzRec*0.5+0.5) * 255)
				if z8Rec > b8 {
					b8 = z8Rec
				}
			}
			z8 := b8

			dst.SetRGBA(x, y, color.RGBA{
				R: 0,        // A_display = 255 - R = 255
				G: g8,       // Y
				B: z8,       // Z
				A: 255 - r8, // X; R_display = 255 - A = X
			})
		}
	}

	return dst
}

// unswizzleNormalMap (Decode): PAA nohq (after DXT5 decode) -> Tangent Space RGB.
// Stored R=0, G=Y, B=Z, A=255-X => X=255-A, Y=G, Z=B.
func unswizzleNormalMap(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewNRGBA(b)

	if n, ok := img.(*image.NRGBA); ok {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			srow := (y - n.Rect.Min.Y) * n.Stride
			drow := (y - dst.Rect.Min.Y) * dst.Stride
			for x := b.Min.X; x < b.Max.X; x++ {
				so := srow + (x-n.Rect.Min.X)*4
				do := drow + (x-dst.Rect.Min.X)*4

				g := n.Pix[so+1]
				bVal := n.Pix[so+2]
				a := n.Pix[so+3]

				rawX := 255 - a
				rawY := g
				rawZ := bVal

				nx := (float64(rawX)/255.0)*2.0 - 1.0
				ny := (float64(rawY)/255.0)*2.0 - 1.0
				nz := (float64(rawZ)/255.0)*2.0 - 1.0

				l := math.Sqrt(nx*nx + ny*ny + nz*nz)
				if l > 0 {
					nx /= l
					ny /= l
					nz /= l
				}

				dst.Pix[do+0] = uint8(clamp01(nx*0.5+0.5) * 255)
				dst.Pix[do+1] = uint8(clamp01(ny*0.5+0.5) * 255)
				dst.Pix[do+2] = uint8(clamp01(nz*0.5+0.5) * 255)
				dst.Pix[do+3] = 255
			}
		}

		return dst
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)

			rawX := 255 - c.A
			rawY := c.G
			rawZ := c.B

			nx := (float64(rawX)/255.0)*2.0 - 1.0
			ny := (float64(rawY)/255.0)*2.0 - 1.0
			nz := (float64(rawZ)/255.0)*2.0 - 1.0

			l := math.Sqrt(nx*nx + ny*ny + nz*nz)
			if l > 0 {
				nx /= l
				ny /= l
				nz /= l
			}

			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(clamp01(nx*0.5+0.5) * 255),
				G: uint8(clamp01(ny*0.5+0.5) * 255),
				B: uint8(clamp01(nz*0.5+0.5) * 255),
				A: 255,
			})
		}
	}

	return dst
}

// clamp01 clamps the value to the range 0-1.
func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// applySwizzlePayload applies a SWIZTAGG payload to an image and returns an NRGBA image.
// Tag bytes are in A, R, G, B order. Each byte encodes:
//
//	bit3: force 0xFF
//	bit2: negate
//	bit1-0: source channel (00=A, 01=R, 10=G, 11=B)
func applySwizzlePayload(img image.Image, tag [4]byte) *image.NRGBA {
	src := toNRGBA(img)
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	get := func(off int, sel uint8) uint8 {
		switch sel & 0x3 {
		case 0:
			return src.Pix[off+3] // A
		case 1:
			return src.Pix[off+0] // R
		case 2:
			return src.Pix[off+1] // G
		default:
			return src.Pix[off+2] // B
		}
	}

	for y := 0; y < b.Dy(); y++ {
		srow := y * src.Stride
		drow := y * dst.Stride
		for x := 0; x < b.Dx(); x++ {
			so := srow + x*4
			do := drow + x*4

			for i := 0; i < 4; i++ {
				t := tag[i]
				var v uint8
				if (t & 0x08) != 0 {
					v = 0xFF
				} else {
					v = get(so, t)
					if (t & 0x04) != 0 {
						v = 255 - v
					}
				}

				switch i {
				case 0:
					dst.Pix[do+3] = v // A
				case 1:
					dst.Pix[do+0] = v // R
				case 2:
					dst.Pix[do+1] = v // G
				case 3:
					dst.Pix[do+2] = v // B
				}
			}
		}
	}

	return dst
}

// applyADSHQSwizzle decodes ADSHQ-style swizzle (tag 02 09 03 09) into a viewable image.
// This maps A -> G (ambient), copies G into B, sets R to 0, and preserves alpha.
func applyADSHQSwizzle(img image.Image) *image.NRGBA {
	src := toNRGBA(img)
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		srow := y * src.Stride
		drow := y * dst.Stride
		for x := 0; x < b.Dx(); x++ {
			so := srow + x*4
			do := drow + x*4
			a := src.Pix[so+3]
			dst.Pix[do+0] = 0 // R
			dst.Pix[do+1] = a // G <- A
			dst.Pix[do+2] = a // B <- G
			dst.Pix[do+3] = a // A
		}
	}

	return dst
}

// toNRGBA converts an image.Image to *image.NRGBA without premultiplication.
func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok {
		return n
	}
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			off := (y-b.Min.Y)*out.Stride + (x-b.Min.X)*4
			out.Pix[off+0] = c.R
			out.Pix[off+1] = c.G
			out.Pix[off+2] = c.B
			out.Pix[off+3] = c.A
		}
	}

	return out
}
