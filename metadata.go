package paa

import (
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

	m := &Metadata{
		Type:       pType,
		Taggs:      tags,
		MipHeaders: make([]MipHeader, 0, 16),
	}

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

		if isDXTPaxType(m.Type) && (w&0x8000) != 0 {
			w &= 0x7FFF
		}

		m.MipHeaders = append(m.MipHeaders, MipHeader{
			Width:  w,
			Height: h,
			Offset: offset,
		})
	}

	return m, nil
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
