// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/woozymasta/bcn"
	"github.com/woozymasta/lzo"
)

// DecodeToDDS decodes a PAA stream and converts it to DDS while preserving mipmaps.
//
// DXT1-5, ARGB1555, and ARGB8 PAA types are supported by this direct conversion path.
func DecodeToDDS(r io.Reader) (*bcn.DDS, error) {
	p, err := DecodePAA(r)
	if err != nil {
		return nil, err
	}

	return p.ToDDS()
}

// DecodeToKTX decodes a PAA stream and converts it to KTX while preserving mipmaps.
//
// DXT1-5, ARGB1555, and ARGB8 PAA types are supported by this direct conversion path.
func DecodeToKTX(r io.Reader) (*bcn.KTX, error) {
	p, err := DecodePAA(r)
	if err != nil {
		return nil, err
	}

	return p.ToKTX()
}

// ToDDS converts decoded PAA data into DDS and preserves all mip levels as-is.
//
// DXT1-5, ARGB1555, and ARGB8 PAA types are exported without re-encoding.
// Other PAA formats require explicit pixel decode + re-encode
// and return ErrUnsupportedFormat here.
func (p *PAA) ToDDS() (*bcn.DDS, error) {
	format, width, height, mips, err := paaDirectMipChain(p)
	if err != nil {
		return nil, err
	}

	return &bcn.DDS{
		Format: format,
		Width:  width,
		Height: height,
		Faces:  []bcn.Face{{Mipmaps: mips}},
	}, nil
}

// ToKTX converts decoded PAA data into KTX and preserves all mip levels as-is.
//
// DXT1-5, ARGB1555, and ARGB8 PAA types are exported without re-encoding.
// Other PAA formats require explicit pixel decode + re-encode
// and return ErrUnsupportedFormat here.
func (p *PAA) ToKTX() (*bcn.KTX, error) {
	format, width, height, mips, err := paaDirectMipChain(p)
	if err != nil {
		return nil, err
	}

	return &bcn.KTX{
		Format: format,
		Width:  width,
		Height: height,
		Faces:  []bcn.Face{{Mipmaps: mips}},
	}, nil
}

// paaDirectMipChain validates and returns directly exportable mip payloads.
func paaDirectMipChain(p *PAA) (bcn.Format, int, int, [][]byte, error) {
	if p == nil {
		return bcn.FormatUnknown, 0, 0, nil, ErrNilPAA
	}

	if len(p.MipMaps) == 0 {
		return bcn.FormatUnknown, 0, 0, nil, ErrNoMipmaps
	}

	format := paxToBcnFormat(p.Type)
	if format == bcn.FormatUnknown {
		return bcn.FormatUnknown, 0, 0, nil, ErrUnsupportedFormat
	}

	base := p.MipMaps[0]
	if base == nil {
		return bcn.FormatUnknown, 0, 0, nil, ErrNilMipMap
	}

	width := int(base.Width)   //nolint:gosec // mip dimensions are uint16 by format.
	height := int(base.Height) //nolint:gosec // mip dimensions are uint16 by format.
	mips := make([][]byte, 0, len(p.MipMaps))

	for _, mm := range p.MipMaps {
		if mm == nil {
			return bcn.FormatUnknown, 0, 0, nil, ErrNilMipMap
		}

		expected := expectedMipSize(p.Type, int(mm.Width), int(mm.Height)) //nolint:gosec // mip dimensions are uint16 by format.
		if expected < 0 {
			return bcn.FormatUnknown, 0, 0, nil, ErrUnsupportedFormat
		}

		if len(mm.Data) != expected {
			return bcn.FormatUnknown, 0, 0, nil, ErrInsufficientData
		}

		mips = append(mips, append([]byte(nil), mm.Data...))
	}

	if len(mips) == 0 {
		return bcn.FormatUnknown, 0, 0, nil, ErrNoMipmaps
	}

	return format, width, height, mips, nil
}

// bcnFormatToPaxType maps a bcn.Format to the corresponding PaxType.
// Returns false for formats not directly representable as PAA (BC4, BC5, uncompressed).
func bcnFormatToPaxType(f bcn.Format) (PaxType, bool) {
	switch f {
	case bcn.FormatDXT1:
		return PaxDXT1, true
	case bcn.FormatDXT3:
		return PaxDXT3, true
	case bcn.FormatDXT5:
		return PaxDXT5, true
	default:
		return 0, false
	}
}

// EncodeFromDDS converts a DDS with BCn-compressed blocks to PAA without re-encoding.
// Only single-face (non-cubemap) DXT1/DXT3/DXT5 textures are supported.
// From opts only UseLZO, WriteGALF/GALFValue, WriteNohqSwizzleTag,
// WriteSwizzleTag/SwizzleTag fields are used; pixel-level options are ignored.
func EncodeFromDDS(w io.Writer, dds *bcn.DDS, opts *EncodeOptions) error {
	if dds == nil {
		return ErrNoMipmaps
	}
	if dds.IsCubemap() {
		return fmt.Errorf("%w: cubemap DDS cannot be converted to PAA", ErrUnsupportedFormat)
	}
	if len(dds.Faces) == 0 || len(dds.Faces[0].Mipmaps) == 0 {
		return ErrNoMipmaps
	}

	paxType, ok := bcnFormatToPaxType(dds.Format)
	if !ok {
		return fmt.Errorf("%w: DDS format %v is not supported by PAA", ErrUnsupportedFormat, dds.Format)
	}

	return encodePAAFromCompressedBlocks(w, paxType, dds.Width, dds.Height, dds.Faces[0].Mipmaps, opts)
}

// EncodeFromKTX converts a KTX with BCn-compressed blocks to PAA without re-encoding.
// Only single-face (non-cubemap) DXT1/DXT3/DXT5 textures are supported.
// From opts only UseLZO, WriteGALF/GALFValue, WriteNohqSwizzleTag,
// WriteSwizzleTag/SwizzleTag fields are used; pixel-level options are ignored.
func EncodeFromKTX(w io.Writer, ktx *bcn.KTX, opts *EncodeOptions) error {
	if ktx == nil {
		return ErrNoMipmaps
	}
	if ktx.IsCubemap() {
		return fmt.Errorf("%w: cubemap KTX cannot be converted to PAA", ErrUnsupportedFormat)
	}
	if len(ktx.Faces) == 0 || len(ktx.Faces[0].Mipmaps) == 0 {
		return ErrNoMipmaps
	}

	paxType, ok := bcnFormatToPaxType(ktx.Format)
	if !ok {
		return fmt.Errorf("%w: KTX format %v is not supported by PAA", ErrUnsupportedFormat, ktx.Format)
	}

	return encodePAAFromCompressedBlocks(w, paxType, ktx.Width, ktx.Height, ktx.Faces[0].Mipmaps, opts)
}

// encodePAAFromCompressedBlocks writes a PAA from pre-compressed BCn mip blocks.
// CGVA is zeroed (pixel data unavailable); CXAM is 0xFF (BI convention for DXT).
func encodePAAFromCompressedBlocks(w io.Writer, paxType PaxType, width, height int, mipmaps [][]byte, opts *EncodeOptions) error {
	if width <= 0 || height <= 0 {
		return ErrInvalidDimensions
	}

	useLZO := opts != nil && opts.UseLZO && isDXT(paxType)
	writeGALF := opts != nil && opts.WriteGALF
	galfValue := byte(1)
	if opts != nil && opts.GALFValue != 0 {
		galfValue = opts.GALFValue
	}
	writeZIWS := false
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
	}

	// Validate block sizes and optionally apply LZO per mip.
	type mipEntry struct {
		data  []byte
		w, h  int
		useLZ bool
	}

	mips := make([]mipEntry, 0, len(mipmaps))
	mw, mh := width, height
	for _, raw := range mipmaps {
		expected := expectedMipSize(paxType, mw, mh)
		if expected < 0 {
			return ErrUnsupportedPixelFmt
		}
		if len(raw) != expected {
			return fmt.Errorf("%w: mip %dx%d: expected %d bytes, got %d", ErrInsufficientData, mw, mh, expected, len(raw))
		}

		data, useLZ := raw, false
		if useLZO {
			var lzoOpts *lzo.CompressOptions
			if opts != nil && opts.LZOLevel > 1 {
				lzoOpts = &lzo.CompressOptions{Level: opts.LZOLevel}
			}
			comp, err := lzo.CompressInto(raw, make([]byte, lzo.MaxCompressedSize(len(raw))), lzoOpts)
			if err == nil && len(comp) < len(raw) {
				data, useLZ = comp, true
			}
		}

		mips = append(mips, mipEntry{data: data, w: mw, h: mh, useLZ: useLZ})
		if mw > 1 {
			mw >>= 1
		}
		if mh > 1 {
			mh >>= 1
		}
	}

	if len(mips) == 0 {
		return ErrNoMipmaps
	}

	// Write PaxType magic.
	if _, err := w.Write(paxType.Bytes()); err != nil {
		return err
	}

	// CGVA: zeros (pixel data not available); CXAM: 0xFF (BI DXT convention).
	if err := writeGGATTag(w, "CGVA", []byte{0, 0, 0, 0}); err != nil {
		return err
	}
	if err := writeGGATTag(w, "CXAM", []byte{0xFF, 0xFF, 0xFF, 0xFF}); err != nil {
		return err
	}
	if writeGALF {
		if err := writeGGATTag(w, "GALF", []byte{galfValue, 0, 0, 0}); err != nil {
			return err
		}
	}
	if writeZIWS {
		if err := writeGGATTag(w, "ZIWS", ziwsTag[:]); err != nil {
			return err
		}
	}

	// Build SFFO offset table (max 16 entries, relative to file start).
	headerSize := 2 + 16 + 16 // paxType + CGVA + CXAM
	if writeGALF {
		headerSize += 16
	}
	if writeZIWS {
		headerSize += 16
	}
	headerSize += 76 + 2 // SFFO tag (12 + 64) + end marker

	var sffo [64]byte
	off := headerSize
	for i := 0; i < len(mips) && i < 16; i++ {
		binary.LittleEndian.PutUint32(sffo[i*4:], uint32(off)) //nolint:gosec // G115: offset fits uint32 for valid PAA files
		off += 2 + 2 + 3 + len(mips[i].data)
	}
	if err := writeGGATTag(w, "SFFO", sffo[:]); err != nil {
		return err
	}
	if _, err := w.Write([]byte{0, 0}); err != nil {
		return err
	}

	// Write mip blocks.
	for _, m := range mips {
		storedW := m.w
		if m.useLZ {
			if m.w > 0x7FFF {
				return ErrInvalidDimensions
			}
			storedW = m.w | 0x8000
		} else if m.w > 0xFFFF {
			return ErrInvalidDimensions
		}
		if m.h > 0xFFFF {
			return ErrInvalidDimensions
		}

		dLen := len(m.data)
		if dLen > 0xFFFFFF {
			return ErrMipDataTooLarge
		}

		dataLen := uint32(dLen) //nolint:gosec // range-checked above.

		if err := writeMipHeader(w, uint16(storedW), uint16(m.h), dataLen); err != nil { //nolint:gosec // range-checked above.
			return err
		}
		if _, err := w.Write(m.data); err != nil {
			return err
		}
	}

	// Trailing padding (matches BI tool output).
	_, err := w.Write([]byte{0, 0, 0, 0, 0, 0})
	return err
}
