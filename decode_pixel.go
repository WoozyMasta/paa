package paa

import (
	"encoding/binary"
	"image"
)

// decodePixelFormat decodes raw uncompressed pixel data into NRGBA.
// Used for non-DXT formats (ARGB8, ARGB1555, ARGB4444, GRAYA).
// PAA stores ARGB8 as BGRA in memory; we swap to R,G,B,A for image.NRGBA.
func decodePixelFormat(format PaxType, rawData []byte, width, height int, img *image.NRGBA) error {
	w, h := width, height

	switch format {
	case PaxARGB8:
		totalPixels := w * h
		if len(rawData) < totalPixels*4 {
			return ErrInsufficientData
		}

		for i := 0; i < totalPixels; i++ {
			off := i * 4
			img.Pix[off+0] = rawData[off+2] // B -> R
			img.Pix[off+1] = rawData[off+1] // G
			img.Pix[off+2] = rawData[off+0] // R -> B
			img.Pix[off+3] = rawData[off+3] // A
		}

	case PaxARGBA5:
		for i := 0; i < len(rawData)/2; i++ {
			p := binary.LittleEndian.Uint16(rawData[i*2:])
			a := uint8(0)
			if (p & 0x8000) != 0 {
				a = 255
			}
			// 5-bit channels, masked so value fits in uint8
			r := uint8((p>>10)&0x1F) << 3 //nolint:gosec // G115
			g := uint8((p>>5)&0x1F) << 3  //nolint:gosec // G115
			b := uint8(p&0x1F) << 3       //nolint:gosec // G115

			off := i * 4
			img.Pix[off+0] = r
			img.Pix[off+1] = g
			img.Pix[off+2] = b
			img.Pix[off+3] = a
		}

	case PaxARGB4:
		for i := 0; i < len(rawData)/2; i++ {
			p := binary.LittleEndian.Uint16(rawData[i*2:])
			// 4-bit channels, masked so value fits in uint8
			img.Pix[i*4+0] = uint8((p & 0x0F) << 4)         //nolint:gosec // G115
			img.Pix[i*4+1] = uint8(((p >> 4) & 0x0F) << 4)  //nolint:gosec // G115
			img.Pix[i*4+2] = uint8(((p >> 8) & 0x0F) << 4)  //nolint:gosec // G115
			img.Pix[i*4+3] = uint8(((p >> 12) & 0x0F) << 4) //nolint:gosec // G115
		}

	case PaxGRAYA:
		totalPixels := w * h
		if len(rawData) < totalPixels*2 {
			return ErrInsufficientData
		}

		for i := 0; i < totalPixels; i++ {
			off := i * 2
			v := rawData[off+0]
			a := rawData[off+1]
			img.Pix[i*4+0] = v
			img.Pix[i*4+1] = v
			img.Pix[i*4+2] = v
			img.Pix[i*4+3] = a
		}

	default:
		return ErrUnsupportedPixelFmt
	}

	return nil
}
