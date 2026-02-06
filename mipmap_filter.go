package paa

import (
	"image"
	"math"

	"github.com/woozymasta/bcn"
	"github.com/woozymasta/paa/texconfig"
)

// generateMipmapsWithFilter generates the mipmaps with the filter applied.
func generateMipmapsWithFilter(img image.Image, useSRGB bool, filter texconfig.MipmapFilter) []image.Image {
	mips := bcn.GenerateMipmaps(img, useSRGB)
	if filter == texconfig.MipmapFilterDefault {
		out := make([]image.Image, len(mips))
		for i := range mips {
			out[i] = mips[i]
		}

		return out
	}

	for level := 1; level < len(mips); level++ {
		applyMipmapFilter(mips[level], level, filter)
	}

	out := make([]image.Image, len(mips))
	for i := range mips {
		out[i] = mips[i]
	}

	return out
}

// applyMipmapFilter applies the mipmap filter to the image.
func applyMipmapFilter(img *image.NRGBA, level int, filter texconfig.MipmapFilter) {
	switch filter {
	case texconfig.MipmapFilterFadeOut:
		fadeOutRGB(img, level)
	case texconfig.MipmapFilterFadeOutAlpha:
		fadeOutAlpha(img, level)
	case texconfig.MipmapFilterAlphaNoise:
		applyAlphaNoise(img, level, 8)
	case texconfig.MipmapFilterAddAlphaNoise:
		applyAlphaNoise(img, level, 16)
	case texconfig.MipmapFilterNormalizeNormalMap:
		normalizeNormalMap(img, false, false, level)
	case texconfig.MipmapFilterNormalizeNormalMapAlpha:
		normalizeNormalMap(img, true, false, level)
	case texconfig.MipmapFilterNormalizeNormalMapNoise:
		normalizeNormalMap(img, true, false, level)
	case texconfig.MipmapFilterNormalizeNormalMapFade:
		normalizeNormalMap(img, true, true, level)
	default:
	}
}

// fadeOutRGB fades out the RGB channels (r, g, b) by a factor.
func fadeOutRGB(img *image.NRGBA, level int) {
	f := math.Pow(0.5, float64(level))
	for y := 0; y < img.Rect.Dy(); y++ {
		row := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			off := row + x*4
			img.Pix[off+0] = blendToMid(img.Pix[off+0], f)
			img.Pix[off+1] = blendToMid(img.Pix[off+1], f)
			img.Pix[off+2] = blendToMid(img.Pix[off+2], f)
		}
	}
}

// fadeOutAlpha fades out the alpha channel.
func fadeOutAlpha(img *image.NRGBA, level int) {
	f := math.Pow(0.5, float64(level))
	for y := 0; y < img.Rect.Dy(); y++ {
		row := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			off := row + x*4
			img.Pix[off+3] = uint8(math.Round(float64(img.Pix[off+3]) * f))
		}
	}
}

// applyAlphaNoise applies the alpha noise to the alpha channel.
func applyAlphaNoise(img *image.NRGBA, level int, strength int) {
	for y := 0; y < img.Rect.Dy(); y++ {
		row := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			off := row + x*4
			noise := alphaNoise(x, y, level, strength)
			img.Pix[off+3] = clampU8(int(img.Pix[off+3]) + noise)
		}
	}
}

// normalizeNormalMap normalizes the normal vector (r, g, b) to a unit vector.
func normalizeNormalMap(img *image.NRGBA, keepAlpha, fade bool, level int) {
	f := math.Pow(0.5, float64(level))
	for y := 0; y < img.Rect.Dy(); y++ {
		row := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			off := row + x*4
			r := img.Pix[off+0]
			g := img.Pix[off+1]
			b := img.Pix[off+2]
			a := img.Pix[off+3]

			nr, ng, nb := normalizeNormal(r, g, b)
			if fade {
				nr = blendToValue(nr, 128, f)
				ng = blendToValue(ng, 128, f)
				nb = blendToValue(nb, 255, f)
			}

			img.Pix[off+0] = nr
			img.Pix[off+1] = ng
			img.Pix[off+2] = nb
			if keepAlpha {
				img.Pix[off+3] = a
			} else {
				img.Pix[off+3] = 255
			}
		}
	}
}

// normalizeNormal normalizes the normal vector (r, g, b) to a unit vector.
func normalizeNormal(r, g, b byte) (byte, byte, byte) {
	nx := (float64(r)/255.0)*2.0 - 1.0
	ny := (float64(g)/255.0)*2.0 - 1.0
	nz := (float64(b)/255.0)*2.0 - 1.0
	l := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if l > 0 {
		nx /= l
		ny /= l
		nz /= l
	} else {
		nx, ny, nz = 0, 0, 1
	}

	return uint8(clamp01(nx*0.5+0.5) * 255),
		uint8(clamp01(ny*0.5+0.5) * 255),
		uint8(clamp01(nz*0.5+0.5) * 255)
}

// blendToMid blends the value to the middle value.
func blendToMid(v byte, factor float64) byte {
	return blendToValue(v, 128, factor)
}

// blendToValue blends the value to the target value.
func blendToValue(v byte, target byte, factor float64) byte {
	return uint8(math.Round(float64(target)*(1.0-factor) + float64(v)*factor))
}

// alphaNoise generates a noise value for the alpha channel.
func alphaNoise(x, y, level, strength int) int {
	n := uint32(x)*1103515245 + uint32(y)*12345 + uint32(level)*1013904223 //nolint:gosec // deterministic noise uses wraparound
	n ^= n >> 13
	n *= 0x85ebca6b
	v := int(n&0x1f) - 16

	if strength <= 0 {
		return v
	}

	return v * strength / 16
}

// clampU8 clamps the value to the range 0-255.
func clampU8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
