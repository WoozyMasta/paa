// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"io"

	"github.com/woozymasta/bcn"
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
