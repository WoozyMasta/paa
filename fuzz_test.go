// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"image"
	"testing"
)

// FuzzDecode checks that decoding arbitrary (possibly malformed) input never
// panics across all read entry points; errors are the expected outcome.
func FuzzDecode(f *testing.F) {
	for _, dims := range [][2]int{{4, 4}, {16, 8}} {
		img := image.NewNRGBA(image.Rect(0, 0, dims[0], dims[1]))
		for i := range img.Pix {
			img.Pix[i] = uint8(i)
		}

		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, img, &EncodeOptions{Type: PaxDXT5, UseLZO: true}); err != nil {
			f.Fatal(err)
		}

		data := buf.Bytes()
		f.Add(append([]byte(nil), data...))
		if len(data) > 8 {
			f.Add(append([]byte(nil), data[:len(data)/2]...)) // truncated
		}
	}
	f.Add([]byte(nil))
	f.Add([]byte("GGAT\x00\x00\x00\x00"))

	dec := NewDecoder()
	f.Fuzz(func(_ *testing.T, data []byte) {
		// None of these must panic on arbitrary input.
		_, _ = Decode(bytes.NewReader(data))
		_, _ = DecodeConfig(bytes.NewReader(data))
		_, _ = DecodeMetadata(bytes.NewReader(data))
		_, _ = DecodeMetadataBytes(data)
		_, _ = DecodePAA(bytes.NewReader(data))
		_, _ = dec.Decode(bytes.NewReader(data))
	})
}

// FuzzEncodeRoundTrip checks that any small image encodes and decodes back to an
// image of the same dimensions without error or panic.
func FuzzEncodeRoundTrip(f *testing.F) {
	f.Add(uint8(8), uint8(8), false, []byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add(uint8(1), uint8(1), true, []byte{255})

	f.Fuzz(func(t *testing.T, w8, h8 uint8, dxt5 bool, pix []byte) {
		w := int(w8)%64 + 1
		h := int(h8)%64 + 1
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		if len(pix) > 0 {
			for i := range img.Pix {
				img.Pix[i] = pix[i%len(pix)]
			}
		}

		typ := PaxDXT1
		if dxt5 {
			typ = PaxDXT5
		}
		noMips := false
		opts := &EncodeOptions{Type: typ, UseLZO: true, GenerateMipmaps: &noMips}

		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, img, opts); err != nil {
			t.Fatalf("encode %dx%d: %v", w, h, err)
		}

		got, err := Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatalf("decode %dx%d: %v", w, h, err)
		}
		if got.Bounds().Dx() != w || got.Bounds().Dy() != h {
			t.Fatalf("dims: got %v want %dx%d", got.Bounds(), w, h)
		}
	})
}
