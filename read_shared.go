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

const (
	// maxPAAInputBytes bounds buffered non-seekable inputs and seekable PAA files.
	// It accommodates a full 8192x8192 RGBA8 texture plus its mip chain.
	maxPAAInputBytes int64 = 512 << 20
	// maxGGATTagBytes prevents metadata allocations disproportionate to PAA content.
	maxGGATTagBytes uint32 = 16 << 20
	// maxGGATTags bounds map growth and scanning work on malformed files.
	maxGGATTags = 64
)

// ensureSeeker wraps non-seekable readers into bytes.Reader.
func ensureSeeker(r io.Reader) (io.Reader, io.Seeker, error) {
	seeker, ok := r.(io.Seeker)
	if ok {
		if rem, known := remainingBytes(r); known && rem > maxPAAInputBytes {
			return nil, nil, ErrInputTooLarge
		}

		return r, seeker, nil
	}

	data, err := readAllLimited(r, maxPAAInputBytes)
	if err != nil {
		return nil, nil, err
	}

	br := bytes.NewReader(data)
	return br, br, nil
}

func readAllLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrInputTooLarge
	}

	return data, nil
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
	tagCount := 0
	for {
		var sig [4]byte
		if _, err := io.ReadFull(r, sig[:]); err != nil {
			return nil, err
		}

		if string(sig[:]) != "GGAT" {
			break
		}
		tagCount++
		if tagCount > maxGGATTags {
			return nil, ErrTooManyTags
		}

		var nameBuf [4]byte
		if _, err := io.ReadFull(r, nameBuf[:]); err != nil {
			return nil, err
		}

		var size uint32
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			return nil, err
		}

		if err := checkGGATTagSize(r, nameBuf[:], size); err != nil {
			return nil, err
		}

		data := make([]byte, size)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}

		tags[string(nameBuf[:])] = data
	}

	return tags, nil
}

// firstMipTags stores only the GGAT data required to decode the first mip.
type firstMipTags struct {
	offsets     [16]uint32
	offsetCount int
	swizzle     [4]byte
	hasSwizzle  bool
}

// readFirstMipTags parses the SFFO offsets and optional ZIWS tag
// without retaining unrelated GGAT payloads.
func readFirstMipTags(r io.Reader) (firstMipTags, error) {
	var tags firstMipTags
	tagCount := 0
	for {
		var sig [4]byte
		if _, err := io.ReadFull(r, sig[:]); err != nil {
			return firstMipTags{}, err
		}
		if sig != [4]byte{'G', 'G', 'A', 'T'} {
			break
		}

		tagCount++
		if tagCount > maxGGATTags {
			return firstMipTags{}, ErrTooManyTags
		}

		var name [4]byte
		if _, err := io.ReadFull(r, name[:]); err != nil {
			return firstMipTags{}, err
		}

		var size uint32
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			return firstMipTags{}, err
		}
		if err := checkGGATTagSize(r, name[:], size); err != nil {
			return firstMipTags{}, err
		}

		switch name {
		case [4]byte{'S', 'F', 'F', 'O'}:
			offsets, count, err := readSFFOOffsetsFixed(r, size)
			if err != nil {
				return firstMipTags{}, err
			}
			tags.offsets = offsets
			tags.offsetCount = count
		case [4]byte{'Z', 'I', 'W', 'S'}:
			ok, err := readTagPrefixAndDiscard(r, size, tags.swizzle[:])
			if err != nil {
				return firstMipTags{}, err
			}
			tags.hasSwizzle = ok
		default:
			if err := discardN(r, int64(size)); err != nil {
				return firstMipTags{}, err
			}
		}
	}

	if tags.offsetCount == 0 {
		return firstMipTags{}, ErrMissingSFFO
	}

	return tags, nil
}

// readSFFOOffsetsFixed returns up to the 16 SFFO offsets defined by the PAA format.
func readSFFOOffsetsFixed(r io.Reader, size uint32) ([16]uint32, int, error) {
	var offsets [16]uint32
	count := 0
	var raw [4]byte
	remaining := size
	for i := 0; i < len(offsets) && remaining >= 4; i++ {
		if _, err := io.ReadFull(r, raw[:]); err != nil {
			return offsets, 0, err
		}
		remaining -= 4

		if offset := binary.LittleEndian.Uint32(raw[:]); offset != 0 && count < len(offsets) {
			offsets[count] = offset
			count++
		}
	}
	if remaining > 0 {
		if err := discardN(r, int64(remaining)); err != nil {
			return offsets, 0, err
		}
	}

	return offsets, count, nil
}

// checkGGATTagSize validates a GGAT payload against configured and input bounds.
func checkGGATTagSize(r io.Reader, name []byte, size uint32) error {
	if size > maxGGATTagBytes {
		return fmt.Errorf("%w: tag %q size %d, limit %d", ErrTagTooLarge, name, size, maxGGATTagBytes)
	}
	if rem, ok := remainingBytes(r); ok && int64(size) > rem {
		return fmt.Errorf("%w: tag %q size %d, %d remain", ErrTagSizeExceedsInput, name, size, rem)
	}

	return nil
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
