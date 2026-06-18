// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Metadata contains lightweight PAA information without mip payload decode.
type Metadata struct {
	// Taggs stores raw GGAT entries by 4-byte key.
	Taggs map[string][]byte
	// MipHeaders stores mip offset and dimensions from SFFO.
	MipHeaders []MipHeader
	// Type is texture pax type from file header.
	Type PaxType
}

// MetadataHeaders contains compact metadata for header-oriented pipelines.
// It avoids retaining full GGAT map and keeps only commonly used tags.
type MetadataHeaders struct {
	// MipHeaders stores mip offset and dimensions from SFFO.
	MipHeaders []MipHeader
	// Type is texture pax type from file header.
	Type PaxType
	// GALF stores GALF flags as little-endian uint32.
	GALF uint32
	// AverageColor stores raw CGVA payload in BGRA byte order.
	AverageColor [4]byte
	// MaxColor stores raw CXAM payload in BGRA byte order.
	MaxColor [4]byte
	// HasAverageColor reports whether CGVA tag is present.
	HasAverageColor bool
	// HasMaxColor reports whether CXAM tag is present.
	HasMaxColor bool
	// HasGALF reports whether GALF tag is present.
	HasGALF bool
}

// MipHeader stores minimum per-mip information for metadata consumers.
type MipHeader struct {
	// Offset is the offset of the mip in the file.
	Offset uint32
	// Height is the height of the mip.
	Height uint16
	// Width is the width of the mip.
	Width uint16
}

// DecodeMetadata reads PAA metadata without decoding mip payload bytes.
//
// It parses PaxType, GGAT tags, and SFFO-referenced mip headers (width/height).
// For DXT formats, width top bit is masked out when LZO flag is present.
func DecodeMetadata(r io.Reader) (*Metadata, error) {
	var err error
	r, seeker, err := ensureSeeker(r)
	if err != nil {
		return nil, err
	}

	pType, err := readPaxType(r)
	if err != nil {
		return nil, err
	}

	tags, err := readGGATTags(r)
	if err != nil {
		return nil, err
	}

	offsets, err := sffoOffsets(tags)
	if err != nil {
		return nil, err
	}

	return buildMetadataFromOffsets(r, seeker, pType, tags, offsets)
}

// DecodeMetadataBytes reads metadata from in-memory PAA data.
func DecodeMetadataBytes(data []byte) (*Metadata, error) {
	return DecodeMetadata(bytes.NewReader(data))
}

// DecodeMetadataHeaders reads compact metadata without allocating full GGAT map.
// It extracts only Type, MipHeaders and commonly used tags: CGVA, CXAM, GALF.
func DecodeMetadataHeaders(r io.Reader) (*MetadataHeaders, error) {
	var err error
	r, seeker, err := ensureSeeker(r)
	if err != nil {
		return nil, err
	}

	pType, err := readPaxType(r)
	if err != nil {
		return nil, err
	}

	var (
		sffo    []byte
		headers = &MetadataHeaders{
			Type:       pType,
			MipHeaders: make([]MipHeader, 0, 16),
		}
	)
	for {
		var sig [4]byte
		if _, err := io.ReadFull(r, sig[:]); err != nil {
			return nil, err
		}
		if string(sig[:]) != "GGAT" {
			break
		}

		var name [4]byte
		if _, err := io.ReadFull(r, name[:]); err != nil {
			return nil, err
		}

		var size uint32
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			return nil, err
		}

		switch string(name[:]) {
		case "SFFO":
			if rem, ok := remainingBytes(r); ok && int64(size) > rem {
				return nil, ErrTagSizeExceedsInput
			}
			sffo = make([]byte, size)
			if _, err := io.ReadFull(r, sffo); err != nil {
				return nil, err
			}
		case "CGVA":
			ok, err := readTagPrefixAndDiscard(r, size, headers.AverageColor[:])
			if err != nil {
				return nil, err
			}
			headers.HasAverageColor = ok
		case "CXAM":
			ok, err := readTagPrefixAndDiscard(r, size, headers.MaxColor[:])
			if err != nil {
				return nil, err
			}
			headers.HasMaxColor = ok
		case "GALF":
			var raw [4]byte
			ok, err := readTagPrefixAndDiscard(r, size, raw[:])
			if err != nil {
				return nil, err
			}
			headers.HasGALF = ok
			if ok {
				headers.GALF = binary.LittleEndian.Uint32(raw[:])
			}
		default:
			if err := discardN(r, int64(size)); err != nil {
				return nil, err
			}
		}
	}

	offsets, err := sffoOffsetsRaw(sffo)
	if err != nil {
		return nil, err
	}

	mipHeaders, err := readMipHeadersByOffsets(r, seeker, pType, offsets)
	if err != nil {
		return nil, err
	}
	headers.MipHeaders = mipHeaders
	return headers, nil
}

// DecodeMetadataHeadersBytes reads compact metadata from in-memory PAA data.
func DecodeMetadataHeadersBytes(data []byte) (*MetadataHeaders, error) {
	return DecodeMetadataHeaders(bytes.NewReader(data))
}

// buildMetadataFromOffsets creates Metadata from already parsed SFFO offsets.
func buildMetadataFromOffsets(r io.Reader, seeker io.Seeker, pType PaxType, tags map[string][]byte, offsets []uint32) (*Metadata, error) {
	headers, err := readMipHeadersByOffsets(r, seeker, pType, offsets)
	if err != nil {
		return nil, err
	}

	return &Metadata{
		Type:       pType,
		Taggs:      tags,
		MipHeaders: headers,
	}, nil
}

// readMipHeadersByOffsets seeks to each mip offset and reads width/height pairs.
func readMipHeadersByOffsets(r io.Reader, seeker io.Seeker, pType PaxType, offsets []uint32) ([]MipHeader, error) {
	headers := make([]MipHeader, 0, len(offsets))
	for _, offset := range offsets {
		if _, err := seeker.Seek(int64(offset), io.SeekStart); err != nil {
			return nil, err
		}

		var w uint16
		if err := binary.Read(r, binary.LittleEndian, &w); err != nil {
			return nil, err
		}

		var h uint16
		if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
			return nil, err
		}

		if w == 0 && h == 0 {
			continue
		}

		if isDXTPaxType(pType) && (w&0x8000) != 0 {
			w &= 0x7FFF
		}

		headers = append(headers, MipHeader{
			Width:  w,
			Height: h,
			Offset: offset,
		})
	}

	return headers, nil
}

// readTagPrefixAndDiscard reads up to len(prefix) bytes and discards the rest.
// It returns true only when prefix bytes were fully populated.
func readTagPrefixAndDiscard(r io.Reader, size uint32, prefix []byte) (bool, error) {
	if size == 0 {
		return false, nil
	}
	n := int(size)
	if n < len(prefix) {
		if err := discardN(r, int64(size)); err != nil {
			return false, err
		}

		return false, nil
	}

	if _, err := io.ReadFull(r, prefix); err != nil {
		return false, err
	}
	if remain := int64(size) - int64(len(prefix)); remain > 0 {
		if err := discardN(r, remain); err != nil {
			return false, err
		}
	}

	return true, nil
}

// discardN drains exactly n bytes from r into io.Discard.
func discardN(r io.Reader, n int64) error {
	if n <= 0 {
		return nil
	}

	_, err := io.CopyN(io.Discard, r, n)
	return err
}

// isDXTPaxType reports whether pax type is DXT-based.
func isDXTPaxType(t PaxType) bool {
	switch t {
	case PaxDXT1, PaxDXT2, PaxDXT3, PaxDXT4, PaxDXT5:
		return true
	default:
		return false
	}
}
