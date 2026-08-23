// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/woozymasta/lzo"
)

func dxtMipFixture(width, height uint16, payload []byte) []byte {
	data := make([]byte, 7+len(payload))
	binary.LittleEndian.PutUint16(data[0:2], width)
	binary.LittleEndian.PutUint16(data[2:4], height)
	data[4] = byte(len(payload))
	data[5] = byte(len(payload) >> 8)
	data[6] = byte(len(payload) >> 16)
	copy(data[7:], payload)

	return data
}

func compressMipFixture(t *testing.T, raw []byte) []byte {
	t.Helper()

	dst := make([]byte, lzo.MaxCompressedSize(len(raw)))
	compressed, err := lzo.CompressInto(raw, dst, nil)
	if err != nil {
		t.Fatalf("lzo.CompressInto() error = %v", err)
	}

	return append([]byte(nil), compressed...)
}

func TestDXTLZOFlagOverridesStoredSize(t *testing.T) {
	fixture := dxtMipFixture(0x8004, 4, []byte("not-lzo!"))

	if _, err := readMipMapWithLimit(bytes.NewReader(fixture), PaxDXT1, maxDecodedMipBytes); !errors.Is(err, ErrLZODecompress) {
		t.Fatalf("readMipMapWithLimit() error = %v, want ErrLZODecompress", err)
	}
	if _, _, err := NewDecoder().readMipImage(bytes.NewReader(fixture), PaxDXT1, nil); !errors.Is(err, ErrLZODecompress) {
		t.Fatalf("Decoder.readMipImage() error = %v, want ErrLZODecompress", err)
	}
}

func TestDXTLZORequiresExactDecodedLength(t *testing.T) {
	payload := compressMipFixture(t, []byte{1, 2, 3, 4})
	fixture := dxtMipFixture(0x8004, 4, payload)

	if _, err := readMipMapWithLimit(bytes.NewReader(fixture), PaxDXT1, maxDecodedMipBytes); !errors.Is(err, ErrLZODecompress) {
		t.Fatalf("readMipMapWithLimit() error = %v, want ErrLZODecompress", err)
	}
	if _, _, err := NewDecoder().readMipImage(bytes.NewReader(fixture), PaxDXT1, nil); !errors.Is(err, ErrLZODecompress) {
		t.Fatalf("Decoder.readMipImage() error = %v, want ErrLZODecompress", err)
	}
}

func TestDXTLZORejectsTrailingInput(t *testing.T) {
	payload := append(compressMipFixture(t, make([]byte, 32)), 0xA5)
	fixture := dxtMipFixture(0x8008, 8, payload)

	for _, tt := range []struct {
		name string
		call func() error
	}{
		{
			name: "full mip reader",
			call: func() error {
				_, err := readMipMapWithLimit(bytes.NewReader(fixture), PaxDXT1, maxDecodedMipBytes)
				return err
			},
		},
		{
			name: "reusable decoder",
			call: func() error {
				_, _, err := NewDecoder().readMipImage(bytes.NewReader(fixture), PaxDXT1, nil)
				return err
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if !errors.Is(err, ErrLZODecompress) || !errors.Is(err, ErrLZOTrailingInput) {
				t.Fatalf("decode() error = %v, want ErrLZODecompress and ErrLZOTrailingInput", err)
			}
		})
	}
}

func TestDXTUnflaggedWrongPayloadSize(t *testing.T) {
	fixture := dxtMipFixture(4, 4, []byte{1, 2, 3, 4})
	if _, err := readMipMapWithLimit(bytes.NewReader(fixture), PaxDXT1, maxDecodedMipBytes); !errors.Is(err, ErrInsufficientData) {
		t.Fatalf("readMipMapWithLimit() error = %v, want ErrInsufficientData", err)
	}
	if _, _, err := NewDecoder().readMipImage(bytes.NewReader(fixture), PaxDXT1, nil); !errors.Is(err, ErrInsufficientData) {
		t.Fatalf("Decoder.readMipImage() error = %v, want ErrInsufficientData", err)
	}
}

func TestDXTRawAndFlaggedLZOMipsDecode(t *testing.T) {
	raw := make([]byte, 32)
	compressed := compressMipFixture(t, raw)

	for _, tt := range []struct {
		name    string
		width   uint16
		payload []byte
	}{
		{name: "raw", width: 8, payload: raw},
		{name: "LZO", width: 0x8008, payload: compressed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture := dxtMipFixture(tt.width, 8, tt.payload)
			mip, err := readMipMapWithLimit(bytes.NewReader(fixture), PaxDXT1, maxDecodedMipBytes)
			if err != nil {
				t.Fatalf("readMipMapWithLimit() error = %v", err)
			}
			if !bytes.Equal(mip.Data, raw) {
				t.Fatal("decoded mip data differs from the original")
			}

			img, ok, err := NewDecoder().readMipImage(bytes.NewReader(fixture), PaxDXT1, nil)
			if err != nil {
				t.Fatalf("Decoder.readMipImage() error = %v", err)
			}
			if !ok || img.Bounds().Dx() != 8 || img.Bounds().Dy() != 8 {
				t.Fatalf("Decoder.readMipImage() = (%v, %t), want 8x8 image", img.Bounds(), ok)
			}
		})
	}
}
