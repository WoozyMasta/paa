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
	"github.com/woozymasta/paa/texconfig"
)

var (
	swizzleBenchSink image.Image
	swizzleBenchSize int
)

// swizzleBenchFixture builds one image and matching PAA outside benchmark timing.
func swizzleBenchFixture(b *testing.B, size int, opts *EncodeOptions) (*image.NRGBA, []byte) {
	b.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x), G: uint8(y), B: uint8(x ^ y), A: uint8((x + y) & 0xff),
			})
		}
	}

	var data bytes.Buffer
	if err := EncodeWithOptions(&data, img, opts); err != nil {
		b.Fatalf("build PAA fixture: %v", err)
	}

	return img, append([]byte(nil), data.Bytes()...)
}

// BenchmarkReusableSwizzleDecode measures reusable DXT5 decode with normal-map,
// generic ZIWS, and no-swizzle paths at representative image sizes.
func BenchmarkReusableSwizzleDecode(b *testing.B) {
	for _, size := range []int{512, 2048} {
		b.Run(fmtBenchSize(size)+"/Nohq", func(b *testing.B) {
			_, data := swizzleBenchFixture(b, size, &EncodeOptions{
				Type: PaxDXT5, NormalMapSwizzle: true, WriteNohqSwizzleTag: true,
				GenerateMipmaps: boolPointer(false),
				BCn:             &bcn.EncodeOptions{QualityLevel: bcn.QualityLevelFast, Workers: 1},
			})
			benchmarkReusableSwizzleDecode(b, data)
		})

		if size != 512 {
			continue
		}

		b.Run(fmtBenchSize(size)+"/Generic", func(b *testing.B) {
			_, data := swizzleBenchFixture(b, size, &EncodeOptions{
				Type: PaxDXT5, WriteSwizzleTag: true,
				SwizzleTag:      [4]byte{0x01, 0x02, 0x03, 0x08},
				GenerateMipmaps: boolPointer(false),
				BCn:             &bcn.EncodeOptions{QualityLevel: bcn.QualityLevelFast, Workers: 1},
			})
			benchmarkReusableSwizzleDecode(b, data)
		})
		b.Run(fmtBenchSize(size)+"/NoSwizzle", func(b *testing.B) {
			_, data := swizzleBenchFixture(b, size, &EncodeOptions{
				Type: PaxDXT5, GenerateMipmaps: boolPointer(false),
				BCn: &bcn.EncodeOptions{QualityLevel: bcn.QualityLevelFast, Workers: 1},
			})
			benchmarkReusableSwizzleDecode(b, data)
		})
	}
}

// benchmarkReusableSwizzleDecode measures steady-state reuse of one Decoder.
func benchmarkReusableSwizzleDecode(b *testing.B, data []byte) {
	dec := NewDecoder()
	reader := bytes.NewReader(data)
	if _, err := dec.Decode(reader); err != nil {
		b.Fatalf("warm up decoder: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		reader.Reset(data)
		img, err := dec.Decode(reader)
		if err != nil {
			b.Fatal(err)
		}
		swizzleBenchSink = img
	}
}

// BenchmarkReusableSwizzleEncode measures steady-state normal-map, generic,
// and non-swizzled encoding using one Encoder.
func BenchmarkReusableSwizzleEncode(b *testing.B) {
	swz := texconfig.ChannelSwizzle{
		R: mustBenchSwizzleExpr("1-A"),
		G: mustBenchSwizzleExpr("R"),
		B: mustBenchSwizzleExpr("G"),
		A: mustBenchSwizzleExpr("1"),
	}
	cases := []struct {
		name string
		opts *EncodeOptions
	}{
		{name: "NormalMap", opts: &EncodeOptions{Type: PaxDXT5, NormalMapSwizzle: true, GenerateMipmaps: boolPointer(false)}},
		{name: "Generic", opts: &EncodeOptions{Type: PaxDXT5, Swizzle: &swz, GenerateMipmaps: boolPointer(false)}},
		{name: "NoSwizzle", opts: &EncodeOptions{Type: PaxDXT5, GenerateMipmaps: boolPointer(false)}},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			img, _ := swizzleBenchFixture(b, 512, tc.opts)
			enc := NewEncoder()
			var out bytes.Buffer
			if err := enc.EncodeWithOptions(&out, img, tc.opts); err != nil {
				b.Fatalf("warm up encoder: %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				out.Reset()
				if err := enc.EncodeWithOptions(&out, img, tc.opts); err != nil {
					b.Fatal(err)
				}

				swizzleBenchSize = out.Len()
			}
		})
	}
}

// fmtBenchSize formats benchmark dimensions without allocating in timed loops.
func fmtBenchSize(size int) string {
	if size == 512 {
		return "512x512"
	}

	return "2048x2048"
}

// boolPointer returns a pointer to a benchmark option value.
func boolPointer(v bool) *bool { return &v }

// mustBenchSwizzleExpr parses a constant benchmark channel expression.
func mustBenchSwizzleExpr(s string) texconfig.SwizzleExpr {
	expr, err := texconfig.ParseSwizzleExpr(s)
	if err != nil {
		panic(err)
	}

	return expr
}
