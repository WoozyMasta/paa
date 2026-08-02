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
// If opts.Type is set (e.g. PaxDXT1, PaxDXT3, PaxDXT5), that format is used;
// otherwise format is chosen by alpha.
// Premultiplied-alpha DXT2 and DXT4 are not supported for encoding.
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

	// image.Image does not guarantee that concurrent At calls are safe.
	// Snapshot non-NRGBA inputs before running the stats scan alongside mip generation.
	if _, ok := img.(*image.NRGBA); !ok {
		statImg = toNRGBA(img)
	}

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
		defer func() { <-statsDone }()
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
	if paxType == PaxDXT2 || paxType == PaxDXT4 {
		return ErrUnsupportedFormat
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
			if bounds.Dx() <= minMipSize || bounds.Dy() <= minMipSize {
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
			var format bcn.Format
			switch paxType {
			case PaxDXT1:
				format = bcn.FormatDXT1
			case PaxDXT3:
				format = bcn.FormatDXT3
			case PaxDXT5:
				format = bcn.FormatDXT5
			default:
				return ErrUnsupportedFormat
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
				var lzoOpts *lzo.CompressOptions
				if opts.LZOLevel > 1 {
					lzoOpts = &lzo.CompressOptions{Level: opts.LZOLevel}
				}
				comp, cerr := lzo.CompressInto(dxt, buf[:required], lzoOpts)
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

			searchLimit := 2048
			if opts != nil && opts.LZSSSearchLimit > 0 {
				searchLimit = opts.LZSSSearchLimit
			}
			comp, cerr := lzss.Compress(raw, &lzss.CompressOptions{
				Checksum:    lzss.ChecksumSigned,
				SearchLimit: searchLimit,
			})
			if cerr != nil {
				return cerr
			}

			// Always store LZSS for non-DXT: official tools compress every mip
			// unconditionally, and TexView expects LZSS even for tiny mips where
			// compressed size exceeds raw size.
			data = comp
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
	if err := writeGGATTag(w, "CGVA", avgTag[:]); err != nil {
		return err
	}
	if err := writeGGATTag(w, "CXAM", maxTag[:]); err != nil {
		return err
	}

	// Write optional GALF and ZIWS tags.
	if writeGALF {
		if meta != nil {
			meta.HasGALF = true
			meta.GALF = uint32(galfValue)
		}
		if err := writeGGATTag(w, "GALF", []byte{galfValue, 0, 0, 0}); err != nil {
			return err
		}
	}
	if writeZIWS {
		if err := writeGGATTag(w, "ZIWS", ziwsTag[:]); err != nil {
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

	var sffo [64]byte
	// Fill offsets for each mip (max 16 entries), relative to file start.
	off := int(baseOffset)
	for i := 0; i < len(mips) && i < 16; i++ {
		binary.LittleEndian.PutUint32(sffo[i*4:i*4+4], uint32(off)) //nolint:gosec // G115
		off += 2 + 2 + 3 + len(mips[i].data)
	}
	if meta != nil {
		offsets, err := sffoOffsetsRaw(sffo[:])
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
	if err := writeGGATTag(w, "SFFO", sffo[:]); err != nil {
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
		// Data length is stored as-is, no LZO flag.
		dLen := len(m.data)
		if dLen > 0xFFFFFF {
			return ErrMipDataTooLarge
		}

		if err := writeMipHeader(w, uint16(storedW), uint16(m.h), dLen); err != nil { //nolint:gosec // range-checked above.
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

// writeGGATTag writes one canonical GGAT tag header and payload.
func writeGGATTag(w io.Writer, name string, payload []byte) error {
	var header [12]byte
	copy(header[0:4], "GGAT")
	copy(header[4:8], name)
	binary.LittleEndian.PutUint32(header[8:12], uint32(len(payload))) //nolint:gosec // PAA tag payload sizes fit uint32.
	if _, err := w.Write(header[:]); err != nil {
		return err
	}

	_, err := w.Write(payload)
	return err
}

// writeMipHeader writes the fixed width, height, and 24-bit payload-size fields.
func writeMipHeader(w io.Writer, width, height uint16, dataLen int) error {
	var header [7]byte
	binary.LittleEndian.PutUint16(header[0:2], width)
	binary.LittleEndian.PutUint16(header[2:4], height)
	header[4] = byte(dataLen)
	header[5] = byte(dataLen >> 8)
	header[6] = byte(dataLen >> 16)
	_, err := w.Write(header[:])
	return err
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
