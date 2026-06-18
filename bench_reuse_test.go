// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/woozymasta/bcn"
)

const benchReuseSize = 512

var (
	benchReuseImg     *image.NRGBA
	benchReuseOpts    *EncodeOptions
	benchReusePAA     []byte
	benchReuseSinkN   int
	benchReuseSinkImg image.Image
)

// benchReuseSetup builds a mip-enabled fixture once: a 512x512 image and its
// encoded PAA, shared by the Encoder/Decoder reuse benchmarks.
func benchReuseSetup(b *testing.B) {
	b.Helper()
	if benchReuseImg != nil {
		return
	}

	img := image.NewNRGBA(image.Rect(0, 0, benchReuseSize, benchReuseSize))
	for y := range benchReuseSize {
		for x := range benchReuseSize {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x),
				G: uint8(y),
				B: uint8(x + y),
				A: uint8(x ^ y),
			})
		}
	}

	yes := true
	opts := &EncodeOptions{
		Type:            PaxDXT5,
		GenerateMipmaps: &yes,
		UseLZO:          true,
		BCn:             &bcn.EncodeOptions{QualityLevel: bcn.QualityLevelFast, Workers: 1},
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, opts); err != nil {
		b.Fatalf("setup encode: %v", err)
	}

	benchReuseImg = img
	benchReuseOpts = opts
	benchReusePAA = append([]byte(nil), buf.Bytes()...)
}

// BenchmarkEncoderReuse compares the package-level encode (allocates per call)
// with a reused Encoder (steady-state near-zero allocations).
func BenchmarkEncoderReuse(b *testing.B) {
	benchReuseSetup(b)
	opts := benchReuseOpts

	b.Run("Package", func(b *testing.B) {
		var buf bytes.Buffer
		buf.Grow(len(benchReusePAA))
		b.ReportAllocs()
		b.SetBytes(int64(len(benchReuseImg.Pix)))
		for range b.N {
			buf.Reset()
			if err := EncodeWithOptions(&buf, benchReuseImg, opts); err != nil {
				b.Fatal(err)
			}
			benchReuseSinkN = buf.Len()
		}
	})

	b.Run("Encoder", func(b *testing.B) {
		enc := NewEncoder()
		var buf bytes.Buffer
		buf.Grow(len(benchReusePAA))
		b.ReportAllocs()
		b.SetBytes(int64(len(benchReuseImg.Pix)))
		for range b.N {
			buf.Reset()
			if err := enc.EncodeWithOptions(&buf, benchReuseImg, opts); err != nil {
				b.Fatal(err)
			}
			benchReuseSinkN = buf.Len()
		}
	})
}

// BenchmarkDecoderReuse compares the package-level decode (allocates per call)
// with a reused Decoder (steady-state near-zero allocations).
func BenchmarkDecoderReuse(b *testing.B) {
	benchReuseSetup(b)
	opts := &DecodeOptions{BCn: &bcn.DecodeOptions{Workers: 1}}

	b.Run("Package", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(benchReusePAA)))
		for range b.N {
			img, err := DecodeWithOptions(bytes.NewReader(benchReusePAA), opts)
			if err != nil {
				b.Fatal(err)
			}
			benchReuseSinkImg = img
		}
	})

	b.Run("Decoder", func(b *testing.B) {
		dec := NewDecoder()
		b.ReportAllocs()
		b.SetBytes(int64(len(benchReusePAA)))
		for range b.N {
			img, err := dec.DecodeWithOptions(bytes.NewReader(benchReusePAA), opts)
			if err != nil {
				b.Fatal(err)
			}
			benchReuseSinkImg = img
		}
	})
}
