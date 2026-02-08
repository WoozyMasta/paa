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

		data := make([]byte, size)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}

		tags[string(nameBuf[:])] = data
	}

	return tags, nil
}

// sffoOffsets returns non-zero offsets from SFFO tag.
func sffoOffsets(tags map[string][]byte) ([]uint32, error) {
	sffo, ok := tags["SFFO"]
	if !ok {
		return nil, ErrMissingSFFO
	}

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
