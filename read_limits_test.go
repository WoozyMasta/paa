// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestReadAllLimited(t *testing.T) {
	data, err := readAllLimited(bytes.NewReader([]byte("1234")), 4)
	if err != nil {
		t.Fatalf("readAllLimited at limit: %v", err)
	}
	if string(data) != "1234" {
		t.Fatalf("data = %q, want %q", data, "1234")
	}

	_, err = readAllLimited(bytes.NewReader([]byte("12345")), 4)
	if !errors.Is(err, ErrInputTooLarge) {
		t.Fatalf("readAllLimited over limit error = %v, want ErrInputTooLarge", err)
	}
}

func TestEnsureSeekerRejectsOversizedInput(t *testing.T) {
	_, _, err := ensureSeeker(&sizedReadSeeker{size: maxPAAInputBytes + 1})
	if !errors.Is(err, ErrInputTooLarge) {
		t.Fatalf("ensureSeeker error = %v, want ErrInputTooLarge", err)
	}
}

func TestDecodeRejectsOversizedGGATTag(t *testing.T) {
	data := paaWithGGATs([]ggatEntry{{name: "LARG", size: maxGGATTagBytes + 1}})

	dec := NewDecoder()
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "Decode", fn: func() error { _, err := Decode(bytes.NewReader(data)); return err }},
		{name: "Decoder", fn: func() error { _, err := dec.Decode(bytes.NewReader(data)); return err }},
		{name: "DecodePAA", fn: func() error { _, err := DecodePAA(bytes.NewReader(data)); return err }},
		{name: "Metadata", fn: func() error { _, err := DecodeMetadata(bytes.NewReader(data)); return err }},
		{name: "MetadataHeaders", fn: func() error { _, err := DecodeMetadataHeaders(bytes.NewReader(data)); return err }},
		{name: "DecoderMetadataHeaders", fn: func() error { _, err := dec.DecodeMetadataHeaders(bytes.NewReader(data)); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, ErrTagTooLarge) {
				t.Fatalf("error = %v, want ErrTagTooLarge", err)
			}
		})
	}
}

func TestDecodeRejectsTooManyGGATTags(t *testing.T) {
	tags := make([]ggatEntry, maxGGATTags+1)
	for i := range tags {
		tags[i].name = "JUNK"
	}

	_, err := DecodePAA(bytes.NewReader(paaWithGGATs(tags)))
	if !errors.Is(err, ErrTooManyTags) {
		t.Fatalf("DecodePAA error = %v, want ErrTooManyTags", err)
	}
}

func TestReadMipMapWithLimitRejectsBeforePayloadAllocation(t *testing.T) {
	var data [7]byte
	binary.LittleEndian.PutUint16(data[0:2], 2)
	binary.LittleEndian.PutUint16(data[2:4], 2)
	data[4], data[5], data[6] = 16, 0, 0

	_, err := readMipMapWithLimit(bytes.NewReader(data[:]), PaxARGB8, 15)
	if !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("readMipMapWithLimit error = %v, want ErrImageTooLarge", err)
	}
}

type ggatEntry struct {
	name string
	size uint32
}

type sizedReadSeeker struct {
	pos  int64
	size int64
}

func (r *sizedReadSeeker) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (r *sizedReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.pos = offset
	case io.SeekCurrent:
		r.pos += offset
	case io.SeekEnd:
		r.pos = r.size + offset
	default:
		return 0, errors.New("invalid whence")
	}

	return r.pos, nil
}

func paaWithGGATs(tags []ggatEntry) []byte {
	var out bytes.Buffer
	out.Write(PaxDXT1.Bytes())
	for _, tag := range tags {
		out.WriteString("GGAT")
		out.WriteString(tag.name)
		var size [4]byte
		binary.LittleEndian.PutUint32(size[:], tag.size)
		out.Write(size[:])
	}
	out.Write([]byte{0, 0, 0, 0})

	return out.Bytes()
}
