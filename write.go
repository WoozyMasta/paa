package paa

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"

	"github.com/woozymasta/bcn"
	"github.com/woozymasta/lzo"
	"github.com/woozymasta/lzss"
	"github.com/woozymasta/paa/texconfig"
)

// Encode writes the image as PAA with a single mip level using default settings:
// DXT5 when the image has any non-opaque alpha, otherwise DXT1.
// For filename-based settings, use EncodeWithTexConfig.
func Encode(w io.Writer, img image.Image) error {
	return EncodeWithOptions(w, img, nil)
}

// EncodeWithOptions writes the image as PAA with a single mip level.
// If opts is nil, behavior is the same as Encode (auto DXT1/DXT5 by alpha).
// If opts.NormalMapSwizzle is true, swizzleNormalMap is applied first and format is DXT5 (for _nohq).
// If opts.Type is set (e.g. PaxDXT1, PaxDXT5), that format is used; otherwise format is chosen by alpha.
func EncodeWithOptions(w io.Writer, img image.Image, opts *EncodeOptions) error {
	statImg := img
	if opts != nil && opts.NormalMapSwizzle {
		img = swizzleNormalMap(img)
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	var avgR, avgG, avgB, avgA uint64
	var maxR, maxG, maxB, maxA uint8
	hasAlpha := false

	// Calculate AVG and MAX colors.
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(statImg.At(x, y)).(color.NRGBA)
			r8, g8, b8, a8 := c.R, c.G, c.B, c.A

			if a8 < 255 {
				hasAlpha = true
			}

			avgR += uint64(r8)
			avgG += uint64(g8)
			avgB += uint64(b8)
			avgA += uint64(a8)

			if r8 > maxR {
				maxR = r8
			}
			if g8 > maxG {
				maxG = g8
			}
			if b8 > maxB {
				maxB = b8
			}
			if a8 > maxA {
				maxA = a8
			}
		}
	}

	pixelCount := uint64(width) * uint64(height) //nolint:gosec // bounds are non-negative
	if pixelCount > 0 {
		avgR /= pixelCount
		avgG /= pixelCount
		avgB /= pixelCount
		avgA /= pixelCount
	}

	var paxType PaxType
	if opts != nil && opts.Type != 0 {
		paxType = opts.Type
	} else if opts != nil && opts.NormalMapSwizzle {
		paxType = PaxDXT5
	} else if hasAlpha {
		paxType = PaxDXT5
	} else {
		paxType = PaxDXT1
	}

	var compressedData []byte
	var err error

	// Mipmap options (defaults mimic BI: full chain down to 4x4).
	generateMips := true
	maxMipCount := 0
	minMipSize := 4
	useSRGB := false
	filter := texconfig.MipmapFilterDefault
	if !isDXT(paxType) {
		minMipSize = 1
	}

	// Apply options.
	if opts != nil {
		if opts.GenerateMipmaps != nil {
			generateMips = *opts.GenerateMipmaps
		}
		if opts.MaxMipCount > 0 {
			maxMipCount = opts.MaxMipCount
		}
		if opts.MinMipSize > 0 {
			minMipSize = opts.MinMipSize
		}
		if opts.UseSRGB {
			useSRGB = true
		}
		if opts.MipmapFilter != nil {
			filter = *opts.MipmapFilter
		}
	}

	// BCn encoder options (quality/refinement/workers).
	var bcnOpts *bcn.EncodeOptions
	if opts != nil && opts.BCn != nil {
		bcnOpts = opts.BCn
	}

	useDXT5 := paxType == PaxDXT5
	writeGALF := false
	galfValue := byte(1)
	writeZIWS := false

	// ZIWS tag is written in canonical order: 0x05, 0x04, 0x02, 0x03 for nohq.
	var ziwsTag [4]byte
	if opts != nil {
		if opts.WriteNohqSwizzleTag {
			writeZIWS = true
			ziwsTag = [4]byte{0x05, 0x04, 0x02, 0x03}
		}
		if opts.WriteSwizzleTag {
			writeZIWS = true
			ziwsTag = opts.SwizzleTag
		}
		if opts.WriteGALF {
			writeGALF = true
			if opts.GALFValue != 0 {
				galfValue = opts.GALFValue
			}
		}
	} else if useDXT5 {
		writeGALF = true
	}

	type mipBlock struct {
		data  []byte
		w, h  int
		useLZ bool
	}
	mips := make([]mipBlock, 0, 8)

	// Generate mipmaps.
	// Build mip chain from the original (unswizzled) image, then swizzle per-mip
	// if required. This keeps CGVA/CXAM consistent with the original content.
	mipImages := []image.Image{statImg}
	if generateMips {
		mipImages = make([]image.Image, 0)

		for _, m := range generateMipmapsWithFilter(statImg, useSRGB, filter) {
			mipImages = append(mipImages, m)
			if maxMipCount > 0 && len(mipImages) >= maxMipCount {
				break
			}

			bounds := m.Bounds()
			if bounds.Dx() <= minMipSize && bounds.Dy() <= minMipSize {
				break
			}
		}
	}

	for _, m := range mipImages {
		encodeImg := m
		if opts != nil && opts.NormalMapSwizzle {
			encodeImg = swizzleNormalMap(m)
		}
		if opts != nil && opts.Swizzle != nil && !opts.SkipSwizzle {
			encodeImg = texconfig.ApplyChannelSwizzle(encodeImg, *opts.Swizzle)
		}

		if isDXT(paxType) {
			if paxType == PaxDXT5 {
				compressedData, _, _, err = bcn.EncodeImageWithOptions(encodeImg, bcn.FormatDXT5, bcnOpts)
			} else {
				compressedData, _, _, err = bcn.EncodeImageWithOptions(encodeImg, bcn.FormatDXT1, bcnOpts)
			}
			if err != nil {
				return err
			}
		} else {
			compressedData, err = encodePixelFormat(paxType, encodeImg)
			if err != nil {
				return err
			}
		}

		// Per-mip LZO: DXT only, use only if it reduces size.
		useLZO := opts != nil && opts.UseLZO && isDXT(paxType)
		useLZ := false
		if useLZO {
			comp, cerr := lzo.Compress(compressedData, nil)
			if cerr != nil {
				return cerr
			}
			if len(comp) < len(compressedData) {
				compressedData = comp
				useLZ = true
			}
		}
		// Non-DXT LZSS: used by BI tools; apply if it reduces size.
		if !isDXT(paxType) {
			comp, cerr := lzss.Compress(compressedData, &lzss.CompressOptions{
				Checksum:    lzss.ChecksumSigned,
				SearchLimit: 2048,
			})
			if cerr != nil {
				return cerr
			}

			forceLZSS := opts != nil && opts.ForceLZSS
			if forceLZSS || len(comp) < len(compressedData) {
				compressedData = comp
			}
		}

		b := encodeImg.Bounds()
		mips = append(mips, mipBlock{
			w:     b.Dx(),
			h:     b.Dy(),
			data:  compressedData,
			useLZ: useLZ,
		})
	}

	// Write PaxType as first tag.
	if _, err := w.Write(paxType.Bytes()); err != nil {
		return err
	}

	// Write tag in canonical order: GGAT, NAME, LEN, DATA.
	writeTag := func(name string, payload []byte) error {
		if _, err := w.Write([]byte("GGAT")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(name)); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, uint32(len(payload))); err != nil { //nolint:gosec // G115
			return err
		}
		if _, err := w.Write(payload); err != nil {
			return err
		}
		return nil
	}

	if opts != nil && opts.NormalMapSwizzle {
		maxR, maxG, maxB, maxA = 255, 255, 255, 255
	}

	// BI tools appear to write CXAM as full 0xFF for DXT textures, regardless of actual max.
	// This affects viewer stats but not pixel payload.
	if (opts != nil && opts.ForceCXAMFull) || (opts == nil && isDXT(paxType)) {
		maxR, maxG, maxB, maxA = 255, 255, 255, 255
	}

	if err := writeTag("CGVA", []byte{uint8(avgB), uint8(avgG), uint8(avgR), uint8(avgA)}); err != nil { //nolint:gosec // G115
		return err
	}
	if err := writeTag("CXAM", []byte{maxB, maxG, maxR, maxA}); err != nil {
		return err
	}

	// Write optional GALF and ZIWS tags.
	if writeGALF {
		if err := writeTag("GALF", []byte{galfValue, 0, 0, 0}); err != nil {
			return err
		}
	}
	if writeZIWS {
		if err := writeTag("ZIWS", ziwsTag[:]); err != nil {
			return err
		}
	}

	// Calculate SFFO offset.
	offset := 2 + 16 + 16
	if writeGALF {
		offset += 16
	}
	if writeZIWS {
		offset += 16
	}
	offset += 76 + 2
	baseOffset := uint32(offset) //nolint:gosec // G115

	sffo := make([]byte, 64)
	// Fill offsets for each mip (max 16 entries), relative to file start.
	off := int(baseOffset)
	for i := 0; i < len(mips) && i < 16; i++ {
		binary.LittleEndian.PutUint32(sffo[i*4:i*4+4], uint32(off)) //nolint:gosec // G115
		off += 2 + 2 + 3 + len(mips[i].data)
	}
	if err := writeTag("SFFO", sffo); err != nil {
		return err
	}

	if _, err := w.Write([]byte{0, 0}); err != nil {
		return err
	}

	for _, m := range mips {
		// LZO is signaled by width's top bit for this mip only.
		if m.w < 0 || m.h < 0 {
			return ErrInvalidDimensions
		}

		// Width is stored with LZO flag if used.
		storedW := m.w
		if m.useLZ {
			if m.w > 0x7fff {
				return ErrInvalidDimensions
			}
			storedW = m.w | 0x8000
		} else if m.w > 0xffff {
			return ErrInvalidDimensions
		}

		// Height is always stored as-is, no LZO flag.
		if m.h > 0xffff {
			return ErrInvalidDimensions
		}
		if err := binary.Write(w, binary.LittleEndian, uint16(storedW)); err != nil { //nolint:gosec // G115
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, uint16(m.h)); err != nil { //nolint:gosec // G115
			return err
		}

		// Data length is stored as-is, no LZO flag.
		dLen := len(m.data)
		if _, err := w.Write([]byte{byte(dLen), byte(dLen >> 8), byte(dLen >> 16)}); err != nil {
			return err
		}

		if _, err := w.Write(m.data); err != nil {
			return err
		}
	}

	// Padding to 64-byte alignment.
	if _, err := w.Write([]byte{0, 0, 0, 0, 0, 0}); err != nil {
		return err
	}

	return nil
}
