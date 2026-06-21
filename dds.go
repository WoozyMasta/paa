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
// Only DXT-based PAA types are supported by this direct conversion path.
func DecodeToDDS(r io.Reader) (*bcn.DDS, error) {
	p, err := DecodePAA(r)
	if err != nil {
		return nil, err
	}

	return p.ToDDS()
}

// DecodeToKTX decodes a PAA stream and converts it to KTX while preserving mipmaps.
//
// Only DXT-based PAA types are supported by this direct conversion path.
func DecodeToKTX(r io.Reader) (*bcn.KTX, error) {
	p, err := DecodePAA(r)
	if err != nil {
		return nil, err
	}

	return p.ToKTX()
}

// ToDDS converts decoded PAA data into DDS and preserves all mip levels as-is.
//
// Only DXT-based PAA types are supported. Non-DXT PAA formats require explicit
// pixel decode + re-encode and return ErrUnsupportedFormat here.
func (p *PAA) ToDDS() (*bcn.DDS, error) {
	format, width, height, mips, err := paaCompressedMipChain(p)
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
// Only DXT-based PAA types are supported. Non-DXT PAA formats require explicit
// pixel decode + re-encode and return ErrUnsupportedFormat here.
func (p *PAA) ToKTX() (*bcn.KTX, error) {
	format, width, height, mips, err := paaCompressedMipChain(p)
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

// paaCompressedMipChain validates and returns BCn-compatible mip payloads.
func paaCompressedMipChain(p *PAA) (bcn.Format, int, int, [][]byte, error) {
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

	width := int(p.MipMaps[0].Width)   //nolint:gosec // mip dimensions are uint16 by format.
	height := int(p.MipMaps[0].Height) //nolint:gosec // mip dimensions are uint16 by format.
	mips := make([][]byte, 0, len(p.MipMaps))

	for _, mm := range p.MipMaps {
		if mm == nil {
			continue
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
			comp, err := lzo.CompressInto(raw, make([]byte, lzo.MaxCompressedSize(len(raw))), nil)
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
		_, err := w.Write(payload)
		return err
	}

	// CGVA: zeros (pixel data not available); CXAM: 0xFF (BI DXT convention).
	if err := writeTag("CGVA", []byte{0, 0, 0, 0}); err != nil {
		return err
	}
	if err := writeTag("CXAM", []byte{0xFF, 0xFF, 0xFF, 0xFF}); err != nil {
		return err
	}
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

	// Build SFFO offset table (max 16 entries, relative to file start).
	headerSize := 2 + 16 + 16 // paxType + CGVA + CXAM
	if writeGALF {
		headerSize += 16
	}
	if writeZIWS {
		headerSize += 16
	}
	headerSize += 76 + 2 // SFFO tag (12 + 64) + end marker

	sffo := make([]byte, 64)
	off := headerSize
	for i := 0; i < len(mips) && i < 16; i++ {
		binary.LittleEndian.PutUint32(sffo[i*4:], uint32(off)) //nolint:gosec // G115: offset fits uint32 for valid PAA files
		off += 2 + 2 + 3 + len(mips[i].data)
	}
	if err := writeTag("SFFO", sffo); err != nil {
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

		if err := binary.Write(w, binary.LittleEndian, uint16(storedW)); err != nil { //nolint:gosec // G115: range-checked above
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, uint16(m.h)); err != nil { //nolint:gosec // G115: range-checked above
			return err
		}

		dLen := len(m.data)
		if dLen > 0xFFFFFF {
			return ErrMipDataTooLarge
		}
		var dLenBuf [4]byte
		binary.LittleEndian.PutUint32(dLenBuf[:], uint32(dLen)) //nolint:gosec // G115: range-checked above
		if _, err := w.Write(dLenBuf[:3]); err != nil {
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
