// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

/*
Package paa reads and writes PAA texture files for Arma and DayZ.

See the [PAA File Format reference] for details about the on-disk format.

[Decode] reads the first mip level as an image;
[DecodePAA] reads the full texture, including tags and mipmaps.
[Encode] uses default settings, while [EncodeWithOptions]
and [EncodeWithTexConfig] allow explicit or filename-based settings.

[Encoder] and [Decoder] reuse buffers across calls.
Use [Encoder.Reset] or [Decoder.Reset] to release retained memory.

For batch processing, use one encoder or decoder per goroutine
and parallelize across textures with one BCn worker per call.
PAA decoding defaults to at most min(GOMAXPROCS, 4) BCn workers
because its LZO step is serial.

To register PAA with [image.Decode] and [image.DecodeConfig],
blank-import [github.com/woozymasta/paa/img].

[PAA File Format reference]: https://community.bistudio.com/wiki/PAA_File_Format
*/
package paa

// PAA represents a PAA texture file (Arma/Bohemia format).
// For use with [image.Decode], blank-import [github.com/woozymasta/paa/img].
type PAA struct {
	// Tag identifiers (e.g. "CGVA", "CXAM", "GALF", "ZIWS", "SFFO") are stored as-is.
	// This package writes tags in the canonical order used by BI tools:
	// CGVA, CXAM, GALF (optional), ZIWS (optional), SFFO.
	Taggs map[string][]byte

	// Mipmaps are stored in the file in order as [MipMap] values.
	MipMaps []*MipMap

	// Palette is stored in the file.
	Palette []byte

	// Type is the [PaxType] of the texture.
	Type PaxType
}
