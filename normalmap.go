// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"image"
	"image/color"
	"math"
)

// swizzleDXT5NM is the SWIZTAGG payload for DXT5 normal maps (_nohq/_nofhq).
var swizzleDXT5NM = [4]byte{0x05, 0x04, 0x02, 0x03}

// swizzleNormalMapInto converts tangent-space RGB (R=X, G=Y, B=Z) to PAA nohq storage channels.
// The engine reconstructs display channels as R=255-A, G=G, B=B, A=255-R,
// so storage is R=0, G=Y, B=Z, A=255-X.
// When Z is lower than the value reconstructed from X and Y, use the reconstructed value.
func swizzleNormalMapInto(dst, src *image.NRGBA) *image.NRGBA {
	bounds := src.Bounds()
	dst = reuseNRGBA(dst, bounds.Dx(), bounds.Dy())

	for y := 0; y < bounds.Dy(); y++ {
		srow := y * src.Stride
		drow := y * dst.Stride

		for x := 0; x < bounds.Dx(); x++ {
			so := srow + x*4
			do := drow + x*4
			r8 := src.Pix[so]
			g8 := src.Pix[so+1]
			b8 := src.Pix[so+2]

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

			dst.Pix[do+0] = 0        // A_display = 255 - R = 255
			dst.Pix[do+1] = g8       // Y
			dst.Pix[do+2] = z8       // Z
			dst.Pix[do+3] = 255 - r8 // X; R_display = 255 - A = X
		}
	}

	return dst
}

// unswizzleNormalMap (Decode): PAA nohq (after DXT5 decode) -> Tangent Space RGB.
// Stored R=0, G=Y, B=Z, A=255-X => X=255-A, Y=G, Z=B.
func unswizzleNormalMap(img image.Image) image.Image {
	n, ok := img.(*image.NRGBA)
	if !ok {
		n = toNRGBA(img)
	}

	return unswizzleNormalMapInPlace(n)
}

// unswizzleNormalMapInPlace reconstructs normalized tangent-space channels.
func unswizzleNormalMapInPlace(img *image.NRGBA) *image.NRGBA {
	b := img.Bounds()

	for y := b.Min.Y; y < b.Max.Y; y++ {
		row := (y - b.Min.Y) * img.Stride
		for x := b.Min.X; x < b.Max.X; x++ {
			off := row + (x-b.Min.X)*4
			// Cache source channels before writing so the operation is safely in place.
			rawX := 255 - img.Pix[off+3]
			rawY := img.Pix[off+1]
			rawZ := img.Pix[off+2]

			nx := (float64(rawX)/255.0)*2.0 - 1.0
			ny := (float64(rawY)/255.0)*2.0 - 1.0
			nz := (float64(rawZ)/255.0)*2.0 - 1.0

			l := math.Sqrt(nx*nx + ny*ny + nz*nz)
			if l > 0 {
				nx /= l
				ny /= l
				nz /= l
			}

			img.Pix[off+0] = uint8(clamp01(nx*0.5+0.5) * 255)
			img.Pix[off+1] = uint8(clamp01(ny*0.5+0.5) * 255)
			img.Pix[off+2] = uint8(clamp01(nz*0.5+0.5) * 255)
			img.Pix[off+3] = 255
		}
	}

	return img
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
	for y := b.Min.Y; y < b.Max.Y; y++ {
		row := (y - b.Min.Y) * src.Stride
		for x := b.Min.X; x < b.Max.X; x++ {
			off := row + (x-b.Min.X)*4
			applyZIWSPixel(src.Pix[off:off+4], tag)
		}
	}

	return src
}

// applyADSHQSwizzle decodes ADSHQ-style swizzle (tag 02 09 03 09) into a viewable image.
// This maps A -> G (ambient), stored G -> B, sets R to 0, and restores opaque alpha.
func applyADSHQSwizzle(img image.Image) *image.NRGBA {
	src := toNRGBA(img)
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		row := (y - b.Min.Y) * src.Stride
		for x := b.Min.X; x < b.Max.X; x++ {
			off := row + (x-b.Min.X)*4
			storedG := src.Pix[off+1]
			storedA := src.Pix[off+3]
			src.Pix[off+0] = 0       // R
			src.Pix[off+1] = storedA // G <- A
			src.Pix[off+2] = storedG // B <- stored G
			src.Pix[off+3] = 255     // A
		}
	}

	return src
}

// applyZIWSPixel transforms one RGBA pixel using A,R,G,B ordered tag bytes.
func applyZIWSPixel(pixel []byte, tag [4]byte) {
	r, g, b, a := pixel[0], pixel[1], pixel[2], pixel[3]
	values := [4]byte{
		ziwsChannelValue(tag[0], r, g, b, a),
		ziwsChannelValue(tag[1], r, g, b, a),
		ziwsChannelValue(tag[2], r, g, b, a),
		ziwsChannelValue(tag[3], r, g, b, a),
	}
	pixel[3], pixel[0], pixel[1], pixel[2] = values[0], values[1], values[2], values[3]
}

// ziwsChannelValue selects, inverts, or supplies one ZIWS channel value.
func ziwsChannelValue(tag, r, g, b, a byte) byte {
	switch tag {
	case 0x08:
		return 255
	case 0x09:
		return 0
	}

	var value byte
	switch tag & 0x03 {
	case 0:
		value = a
	case 1:
		value = r
	case 2:
		value = g
	case 3:
		value = b
	}

	if tag&0x04 != 0 {
		return 255 - value
	}

	return value
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
