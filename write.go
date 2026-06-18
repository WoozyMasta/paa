// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

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
// If opts.NormalMapSwizzle is true, normal-map swizzle is applied per mip and format is DXT5 (for _nohq).
// If opts.Type is set (e.g. PaxDXT1, PaxDXT5), that format is used; otherwise format is chosen by alpha.
func EncodeWithOptions(w io.Writer, img image.Image, opts *EncodeOptions) error {
	return (&Encoder{}).encodeWithOptions(w, img, opts, nil)
}

// EncodeWithOptionsAndMetadataHeaders writes the image as PAA and returns
// compact metadata headers collected during the same encode pass.
func EncodeWithOptionsAndMetadataHeaders(w io.Writer, img image.Image, opts *EncodeOptions) (*MetadataHeaders, error) {
	return (&Encoder{}).EncodeWithOptionsAndMetadataHeaders(w, img, opts)
}

// encodeWithOptions performs the full encode flow and optionally fills metadata,
// reusing the Encoder's buffers across calls.
func (e *Encoder) encodeWithOptions(w io.Writer, img image.Image, opts *EncodeOptions, meta *MetadataHeaders) error {
	statImg := img

	// imageStats only feeds the CGVA/CXAM tags, which are written after the heavier mip + BCn + LZO pipeline.
	// When the output type does not depend on hasAlpha,
	// run the (serial) stats scan concurrently with that pipeline and join before the tags.
	var (
		avgR, avgG, avgB, avgA uint64
		maxR, maxG, maxB, maxA uint8
		hasAlpha               bool
		statsDone              chan struct{}
	)
	computeStats := func() {
		avgR, avgG, avgB, avgA, maxR, maxG, maxB, maxA, hasAlpha = imageStats(statImg)
	}
	if opts != nil && (opts.Type != 0 || opts.NormalMapSwizzle) {
		statsDone = make(chan struct{})
		go func() {
			computeStats()
			close(statsDone)
		}()
	} else {
		// hasAlpha is needed now to choose the type.
		computeStats()
	}

	var paxType PaxType
	switch {
	case opts != nil && opts.Type != 0:
		paxType = opts.Type
	case opts != nil && opts.NormalMapSwizzle:
		paxType = PaxDXT5
	case hasAlpha:
		paxType = PaxDXT5
	default:
		paxType = PaxDXT1
	}
	if meta != nil {
		*meta = MetadataHeaders{
			Type:       paxType,
			MipHeaders: make([]MipHeader, 0, 16),
		}
	}

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

	// Generate the mip chain into the reusable pool,
	// then apply the optional per-level filter in place.
	// The chain is built from the original (unswizzled) image;
	// per-mip swizzle (if any) is applied below.
	// This keeps CGVA/CXAM consistent with the original content.
	mipImages := e.mipImages[:0]
	if generateMips {
		e.mipPool = bcn.GenerateMipmapsInto(e.mipPool, statImg, maxMipCount, useSRGB)
		if filter != texconfig.MipmapFilterDefault {
			for level := 1; level < len(e.mipPool); level++ {
				applyMipmapFilter(e.mipPool[level], level, filter)
			}
		}

		for _, m := range e.mipPool {
			mipImages = append(mipImages, m)
			bounds := m.Bounds()
			if bounds.Dx() <= minMipSize && bounds.Dy() <= minMipSize {
				break
			}
		}
	} else {
		mipImages = append(mipImages, statImg)
	}
	e.mipImages = mipImages

	e.mips = e.mips[:0]
	for i, m := range mipImages {
		encodeImg := m
		if opts != nil && opts.NormalMapSwizzle {
			encodeImg = swizzleNormalMap(m)
		}
		if opts != nil && opts.Swizzle != nil && !opts.SkipSwizzle {
			encodeImg = texconfig.ApplyChannelSwizzle(encodeImg, *opts.Swizzle)
		}

		var data []byte
		useLZ := false

		if isDXT(paxType) {
			format := bcn.FormatDXT1
			if paxType == PaxDXT5 {
				format = bcn.FormatDXT5
			}

			var encErr error
			e.dxtBuf, _, _, encErr = bcn.EncodeImageInto(e.dxtBuf, encodeImg, format, bcnOpts)
			if encErr != nil {
				return encErr
			}
			dxt := e.dxtBuf

			// Retain each mip's payload in its own reusable buffer; the transient
			// dxtBuf is overwritten by the next mip. Per-mip LZO is used only when
			// it reduces size, otherwise the raw DXT is retained.
			buf := e.payload(i)
			if opts != nil && opts.UseLZO {
				required := lzo.MaxCompressedSize(len(dxt))
				if cap(buf) < required {
					buf = make([]byte, required)
				}
				comp, cerr := lzo.CompressInto(dxt, buf[:required], nil)
				if cerr != nil {
					return cerr
				}
				if len(comp) < len(dxt) {
					data = comp
					useLZ = true
				}
			}
			if !useLZ {
				if cap(buf) < len(dxt) {
					buf = make([]byte, len(dxt))
				}
				buf = buf[:len(dxt)]
				copy(buf, dxt)
				data = buf
			}
			e.payloads[i] = buf
		} else {
			// Non-DXT (rare): encode raw pixels and LZSS-compress. These paths
			// allocate per mip as before.
			raw, perr := encodePixelFormat(paxType, encodeImg)
			if perr != nil {
				return perr
			}

			comp, cerr := lzss.Compress(raw, &lzss.CompressOptions{
				Checksum:    lzss.ChecksumSigned,
				SearchLimit: 2048,
			})
			if cerr != nil {
				return cerr
			}

			if (opts != nil && opts.ForceLZSS) || len(comp) < len(raw) {
				data = comp
			} else {
				data = raw
			}
		}

		b := encodeImg.Bounds()
		e.mips = append(e.mips, mipBlock{
			w:     b.Dx(),
			h:     b.Dy(),
			data:  data,
			useLZ: useLZ,
		})
	}
	mips := e.mips

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

	// Join the concurrent stats scan before reading/overriding avg*/max*.
	if statsDone != nil {
		<-statsDone
	}

	if opts != nil && opts.NormalMapSwizzle {
		maxR, maxG, maxB, maxA = 255, 255, 255, 255
	}

	// BI tools appear to write CXAM as full 0xFF for DXT textures, regardless of actual max.
	// This affects viewer stats but not pixel payload.
	if (opts != nil && opts.ForceCXAMFull) || (opts == nil && isDXT(paxType)) {
		maxR, maxG, maxB, maxA = 255, 255, 255, 255
	}

	avgTag := [4]byte{uint8(avgB), uint8(avgG), uint8(avgR), uint8(avgA)} //nolint:gosec // stats are uint8-range by construction.
	maxTag := [4]byte{maxB, maxG, maxR, maxA}
	if meta != nil {
		meta.HasAverageColor = true
		meta.AverageColor = avgTag
		meta.HasMaxColor = true
		meta.MaxColor = maxTag
	}
	if err := writeTag("CGVA", avgTag[:]); err != nil {
		return err
	}
	if err := writeTag("CXAM", maxTag[:]); err != nil {
		return err
	}

	// Write optional GALF and ZIWS tags.
	if writeGALF {
		if meta != nil {
			meta.HasGALF = true
			meta.GALF = uint32(galfValue)
		}
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
	if meta != nil {
		offsets, err := sffoOffsetsRaw(sffo)
		if err != nil {
			return err
		}

		for i, offset := range offsets {
			if i >= len(mips) {
				break
			}
			if mips[i].w < 0 || mips[i].w > 0xFFFF || mips[i].h < 0 || mips[i].h > 0xFFFF {
				return ErrInvalidDimensions
			}
			// #nosec G115 -- dimensions are range-checked to uint16 bounds above.
			w16 := uint16(mips[i].w)
			// #nosec G115 -- dimensions are range-checked to uint16 bounds above.
			h16 := uint16(mips[i].h)

			meta.MipHeaders = append(meta.MipHeaders, MipHeader{
				Offset: offset,
				Width:  w16,
				Height: h16,
			})
		}
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
		if dLen > 0xFFFFFF {
			return ErrMipDataTooLarge
		}

		var dLenBuf [4]byte
		// #nosec G115 -- dLen is range-checked to 24-bit above.
		binary.LittleEndian.PutUint32(dLenBuf[:], uint32(dLen))
		if _, err := w.Write(dLenBuf[:3]); err != nil {
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

// imageStats calculates per-image average and maximum channel values.
func imageStats(img image.Image) (avgR, avgG, avgB, avgA uint64, maxR, maxG, maxB, maxA uint8, hasAlpha bool) {
	if nrgba, ok := img.(*image.NRGBA); ok {
		return nrgbaImageStats(nrgba)
	}

	bounds := img.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
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

	pixelCount := uint64(bounds.Dx()) * uint64(bounds.Dy()) //nolint:gosec // bounds are non-negative
	if pixelCount > 0 {
		avgR /= pixelCount
		avgG /= pixelCount
		avgB /= pixelCount
		avgA /= pixelCount
	}

	return avgR, avgG, avgB, avgA, maxR, maxG, maxB, maxA, hasAlpha
}

// nrgbaImageStats is a zero-allocation fast path for *image.NRGBA input.
func nrgbaImageStats(img *image.NRGBA) (avgR, avgG, avgB, avgA uint64, maxR, maxG, maxB, maxA uint8, hasAlpha bool) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, false
	}

	rowOffsetX := (bounds.Min.X - img.Rect.Min.X) * 4

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		rowStart := (y-img.Rect.Min.Y)*img.Stride + rowOffsetX
		row := img.Pix[rowStart : rowStart+width*4]

		for i := 0; i < len(row); i += 4 {
			r8, g8, b8, a8 := row[i], row[i+1], row[i+2], row[i+3]

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
	avgR /= pixelCount
	avgG /= pixelCount
	avgB /= pixelCount
	avgA /= pixelCount

	return avgR, avgG, avgB, avgA, maxR, maxG, maxB, maxA, hasAlpha
}
