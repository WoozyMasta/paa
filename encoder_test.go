// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func encoderTestImage(w, h int, alpha bool) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			a := uint8(255)
			if alpha {
				a = uint8((x ^ y) & 0xFF)
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 7),
				G: uint8(y * 5),
				B: uint8((x + y) * 3),
				A: a,
			})
		}
	}
	return img
}

// TestEncoderMatchesPackage verifies the reusable Encoder produces byte-identical output
// to the package-level EncodeWithOptions, including when one Encoder
// is reused across images of different sizes and formats (catches buffer-reuse bugs).
func TestEncoderMatchesPackage(t *testing.T) {
	imgs := []*image.NRGBA{
		encoderTestImage(64, 64, true),
		encoderTestImage(40, 24, false),
		encoderTestImage(16, 16, true),
	}

	yes, no := true, false
	cases := []*EncodeOptions{
		{Type: PaxDXT5, UseLZO: true, GenerateMipmaps: &yes},
		{Type: PaxDXT1, UseLZO: true, GenerateMipmaps: &yes},
		{Type: PaxDXT5, UseLZO: true, GenerateMipmaps: &no},
		nil,
	}

	enc := NewEncoder()
	for ci, opts := range cases {
		for ii, img := range imgs {
			var want, got bytes.Buffer
			if err := EncodeWithOptions(&want, img, opts); err != nil {
				t.Fatalf("case %d image %d: package encode: %v", ci, ii, err)
			}
			if err := enc.EncodeWithOptions(&got, img, opts); err != nil {
				t.Fatalf("case %d image %d: encoder encode: %v", ci, ii, err)
			}
			if !bytes.Equal(want.Bytes(), got.Bytes()) {
				t.Fatalf("case %d image %d: Encoder output differs from package EncodeWithOptions", ci, ii)
			}
		}
	}
}

// TestEncoderReuseStable verifies that re-encoding the same image through
// a reused Encoder (after encoding other sizes) yields identical bytes each time.
func TestEncoderReuseStable(t *testing.T) {
	a := encoderTestImage(48, 48, true)
	b := encoderTestImage(20, 36, true)
	opts := &EncodeOptions{Type: PaxDXT5, UseLZO: true}

	enc := NewEncoder()
	var first bytes.Buffer
	if err := enc.EncodeWithOptions(&first, a, opts); err != nil {
		t.Fatal(err)
	}

	var other bytes.Buffer
	if err := enc.EncodeWithOptions(&other, b, opts); err != nil {
		t.Fatal(err)
	}

	var again bytes.Buffer
	if err := enc.EncodeWithOptions(&again, a, opts); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first.Bytes(), again.Bytes()) {
		t.Fatal("re-encoding the same image through a reused Encoder produced different bytes")
	}
}
