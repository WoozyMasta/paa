// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"image"
	"testing"
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
