// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"errors"
	"image"
	"testing"

	"github.com/woozymasta/lzss"
)

func encodePAA(t *testing.T, img *image.NRGBA, opts *EncodeOptions) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, opts); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return append([]byte(nil), buf.Bytes()...)
}

// TestDecoderMatchesPackage verifies the reusable Decoder produces pixel-identical output
// to the package-level Decode, including when reused across sizes/formats.
func TestDecoderMatchesPackage(t *testing.T) {
	imgs := []*image.NRGBA{
		encoderTestImage(64, 64, true),
		encoderTestImage(40, 24, false),
		encoderTestImage(16, 16, true),
	}

	yes, no := true, false
	cases := []*EncodeOptions{
		{Type: PaxDXT5, UseLZO: true, GenerateMipmaps: &no},
		{Type: PaxDXT1, UseLZO: true, GenerateMipmaps: &yes},
		{Type: PaxARGB8, GenerateMipmaps: &yes},
		{Type: PaxARGBA5, GenerateMipmaps: &yes},
	}

	dec := NewDecoder()
	for ci, opts := range cases {
		for ii, src := range imgs {
			data := encodePAA(t, src, opts)

			want, err := Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("case %d image %d: package decode: %v", ci, ii, err)
			}
			got, err := dec.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("case %d image %d: decoder decode: %v", ci, ii, err)
			}

			wn, gn := want.(*image.NRGBA), got.(*image.NRGBA)
			if wn.Rect != gn.Rect {
				t.Fatalf("case %d image %d: rect %v != %v", ci, ii, gn.Rect, wn.Rect)
			}
			if !bytes.Equal(wn.Pix, gn.Pix) {
				t.Fatalf("case %d image %d: Decoder output differs from package Decode", ci, ii)
			}
		}
	}
}

func TestDecoderReuseRejectsTrailingLZSSInput(t *testing.T) {
	data := encodePAA(t, encoderTestImage(8, 8, true), &EncodeOptions{
		Type: PaxARGBA5, GenerateMipmaps: ptrBool(false),
	})
	off := firstMipOffset(parseTagg(data))
	if off == 0 || off+7 > len(data) {
		t.Fatal("invalid first mip offset")
	}
	stored := int(data[off+4]) | int(data[off+5])<<8 | int(data[off+6])<<16
	if stored == 8*8*2 {
		t.Fatal("fixture was not LZSS-compressed")
	}
	data = append(data, 0xA5)
	stored++
	data[off+4] = byte(stored)
	data[off+5] = byte(stored >> 8)
	data[off+6] = byte(stored >> 16)

	dec := NewDecoder()
	if _, err := dec.Decode(
		bytes.NewReader(data),
	); !errors.Is(err, ErrLZSSDecompress) || !errors.Is(err, lzss.ErrTrailingData) {
		t.Fatalf("Decoder.Decode() error = %v, want ErrLZSSDecompress and lzss.ErrTrailingData", err)
	}
}

// TestDecoderReuseStable verifies that re-decoding the same stream through a reused Decoder
// (after decoding a different size) yields identical pixels.
func TestDecoderReuseStable(t *testing.T) {
	opts := &EncodeOptions{Type: PaxDXT5, UseLZO: true}
	da := encodePAA(t, encoderTestImage(48, 48, true), opts)
	db := encodePAA(t, encoderTestImage(20, 36, true), opts)

	dec := NewDecoder()

	r1, err := dec.Decode(bytes.NewReader(da))
	if err != nil {
		t.Fatal(err)
	}
	first := append([]byte(nil), r1.(*image.NRGBA).Pix...)

	if _, err := dec.Decode(bytes.NewReader(db)); err != nil {
		t.Fatal(err)
	}

	r3, err := dec.Decode(bytes.NewReader(da))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, r3.(*image.NRGBA).Pix) {
		t.Fatal("re-decoding the same stream through a reused Decoder produced different pixels")
	}
}
