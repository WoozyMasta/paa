// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// ensureSeeker wraps non-seekable readers into bytes.Reader.
func ensureSeeker(r io.Reader) (io.Reader, io.Seeker, error) {
	seeker, ok := r.(io.Seeker)
	if ok {
		return r, seeker, nil
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	br := bytes.NewReader(data)
	return br, br, nil
}

// readPaxType reads and validates first 2 bytes as pax type.
func readPaxType(r io.Reader) (PaxType, error) {
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, err
	}

	pType, ok := PaxTypeFromBytes(header[:])
	if !ok {
		return 0, ErrInvalidMagic
	}

	return pType, nil
}

// maxDecodedMipBytes caps a single decoded mip (width*height*4)
// to guard against malformed dimensions triggering huge allocations.
// 512 MiB leaves headroom over the practical PAA maximum (8192x8192 RGBA8 = 256 MiB).
const maxDecodedMipBytes = 512 << 20

// remainingBytes reports how many bytes remain to be read when r is seekable.
// All decode paths wrap their reader with ensureSeeker first,
// so this lets size fields from untrusted input be validated against the actual stream length.
func remainingBytes(r io.Reader) (int64, bool) {
	s, ok := r.(io.Seeker)
	if !ok {
		return 0, false
	}

	cur, err := s.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, false
	}
	end, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, false
	}
	if _, err := s.Seek(cur, io.SeekStart); err != nil {
		return 0, false
	}

	return end - cur, true
}

// checkMipBudget rejects mip dimensions whose decoded size would exceed maxDecodedMipBytes,
// guarding against malformed headers.
func checkMipBudget(width, height int) error {
	if int64(width)*int64(height)*4 > maxDecodedMipBytes {
		return ErrImageTooLarge
	}

	return nil
}

// readGGATTags parses all GGAT tags into map.
func readGGATTags(r io.Reader) (map[string][]byte, error) {
	tags := make(map[string][]byte, 8)
	for {
		var sig [4]byte
		if _, err := io.ReadFull(r, sig[:]); err != nil {
			return nil, err
		}

		if string(sig[:]) != "GGAT" {
			break
		}

		var nameBuf [4]byte
		if _, err := io.ReadFull(r, nameBuf[:]); err != nil {
			return nil, err
		}

		var size uint32
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			return nil, err
		}

		// Reject tag sizes larger than the remaining stream before allocating.
		if rem, ok := remainingBytes(r); ok && int64(size) > rem {
			return nil, fmt.Errorf("%w: tag %q size %d, %d remain", ErrTagSizeExceedsInput, nameBuf[:], size, rem)
		}

		data := make([]byte, size)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}

		tags[string(nameBuf[:])] = data
	}

	return tags, nil
}

// sffoOffsets returns non-zero offsets from SFFO tag.
// SFFO is the PAA mip offset table tag ("OFFS", offset list).
func sffoOffsets(tags map[string][]byte) ([]uint32, error) {
	sffo, ok := tags["SFFO"]
	if !ok {
		return nil, ErrMissingSFFO
	}

	return sffoOffsetsRaw(sffo)
}

// sffoOffsetsRaw returns non-zero offsets from raw SFFO payload.
func sffoOffsetsRaw(sffo []byte) ([]uint32, error) {
	out := make([]uint32, 0, len(sffo)/4)
	for i := 0; i+4 <= len(sffo); i += 4 {
		v := binary.LittleEndian.Uint32(sffo[i : i+4])
		if v == 0 {
			continue
		}

		out = append(out, v)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no non-zero offsets", ErrMissingSFFO)
	}

	return out, nil
}
