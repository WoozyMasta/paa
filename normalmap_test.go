// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/woozymasta/paa/texconfig"
)

func TestSwizzleNormalMapPreservesStraightAlpha(t *testing.T) {
	src := image.NewNRGBA(image.Rect(7, 11, 8, 12))
	src.SetNRGBA(7, 11, color.NRGBA{R: 128, G: 128, B: 255, A: 255})

	got := swizzleNormalMapInto(nil, src)
	if got.Bounds() != image.Rect(0, 0, 1, 1) {
		t.Fatalf("swizzled bounds = %v, want (0,0)-(1,1)", got.Bounds())
	}

	want := color.NRGBA{R: 0, G: 128, B: 255, A: 127}
	if pixel := got.NRGBAAt(0, 0); pixel != want {
		t.Fatalf("swizzled pixel = %#v, want %#v", pixel, want)
	}
	if pixel := color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA); pixel != want {
		t.Fatalf("converted pixel = %#v, want %#v", pixel, want)
	}
}

func TestNormalMapEncodeDecodeQuality(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 128, G: 128, B: 255, A: 255})
		}
	}

	noMips := false
	opts := &EncodeOptions{
		Type:                PaxDXT5,
		NormalMapSwizzle:    true,
		WriteNohqSwizzleTag: true,
		GenerateMipmaps:     &noMips,
	}

	var encoded bytes.Buffer
	if err := EncodeWithOptions(&encoded, src, opts); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}

	decoded, err := Decode(&encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := imagePSNR(src, decoded); got < minPSNR {
		t.Fatalf("normal-map round-trip PSNR = %.2f dB, want >= %.2f dB", got, minPSNR)
	}
}

func TestApplySwizzlePayloadDistinguishesConstantSelectors(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 40})

	got := applySwizzlePayload(src, [4]byte{0x09, 0x08, 0x09, 0x08})
	want := color.NRGBA{R: 255, G: 0, B: 255, A: 0}
	if pixel := got.NRGBAAt(0, 0); pixel != want {
		t.Fatalf("swizzled pixel = %#v, want %#v", pixel, want)
	}
}

func TestApplyADSHQSwizzleReconstructsChannels(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 70, B: 0, A: 30})

	got := applyADSHQSwizzle(src)
	want := color.NRGBA{R: 0, G: 30, B: 70, A: 255}
	if pixel := got.NRGBAAt(0, 0); pixel != want {
		t.Fatalf("ADSHQ pixel = %#v, want %#v", pixel, want)
	}
}

// TestApplySwizzlePayloadSupportsInPlacePermutation covers aliased source and destination.
func TestApplySwizzlePayloadSupportsInPlacePermutation(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 40})

	got := applySwizzlePayload(src, [4]byte{0x01, 0x02, 0x03, 0x00})
	if got != src {
		t.Fatal("swizzle allocated instead of transforming the NRGBA image in place")
	}
	if pixel := got.NRGBAAt(0, 0); pixel != (color.NRGBA{R: 20, G: 30, B: 40, A: 10}) {
		t.Fatalf("swizzled pixel = %#v, want %#v", pixel, color.NRGBA{R: 20, G: 30, B: 40, A: 10})
	}
}

// TestApplyChannelSwizzleInPlaceMatchesPublicHelper checks encode swizzle semantics.
func TestApplyChannelSwizzleInPlaceMatchesPublicHelper(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
	src.SetNRGBA(1, 0, color.NRGBA{R: 50, G: 60, B: 70, A: 80})
	swz := texconfig.ChannelSwizzle{
		R: mustTestSwizzleExpr(t, "1-A"), G: mustTestSwizzleExpr(t, "R"),
		B: mustTestSwizzleExpr(t, "G"), A: mustTestSwizzleExpr(t, "1"),
	}
	want := texconfig.ApplyChannelSwizzle(src, swz)

	applyChannelSwizzleInPlace(src, swz)
	if !bytes.Equal(src.Pix, want.Pix) {
		t.Fatalf("in-place pixels = %v, want %v", src.Pix, want.Pix)
	}
}

// TestSwizzleReuseAcrossModes verifies reusable transforms do not leak old pixels.
func TestSwizzleReuseAcrossModes(t *testing.T) {
	swz := texconfig.ChannelSwizzle{
		R: mustTestSwizzleExpr(t, "1-A"), G: mustTestSwizzleExpr(t, "R"),
		B: mustTestSwizzleExpr(t, "G"), A: mustTestSwizzleExpr(t, "1"),
	}
	cases := []struct {
		name string
		size int
		opts *EncodeOptions
	}{
		{
			name: "normal",
			size: 8,
			opts: &EncodeOptions{Type: PaxDXT5,
				NormalMapSwizzle:    true,
				WriteNohqSwizzleTag: true,
			},
		},
		{
			name: "generic",
			size: 4,
			opts: &EncodeOptions{
				Type:            PaxDXT5,
				Swizzle:         &swz,
				WriteSwizzleTag: true,
				SwizzleTag:      [4]byte{0x01, 0x02, 0x03, 0x08},
			},
		},
		{
			name: "none",
			size: 8,
			opts: &EncodeOptions{Type: PaxDXT5},
		},
	}
	noMips := false
	for i := range cases {
		cases[i].opts.GenerateMipmaps = &noMips
	}

	enc := NewEncoder()
	dec := NewDecoder()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := encoderTestImage(tc.size, tc.size, true)
			var got bytes.Buffer
			if err := enc.EncodeWithOptions(&got, src, tc.opts); err != nil {
				t.Fatalf("reused encode: %v", err)
			}
			want := encodePAA(t, src, tc.opts)
			if !bytes.Equal(got.Bytes(), want) {
				t.Fatal("reused encoder output differs from package-level encoding")
			}

			gotImg, err := dec.Decode(bytes.NewReader(got.Bytes()))
			if err != nil {
				t.Fatalf("reused decode: %v", err)
			}
			wantImg, err := Decode(bytes.NewReader(got.Bytes()))
			if err != nil {
				t.Fatalf("package decode: %v", err)
			}
			if !bytes.Equal(gotImg.(*image.NRGBA).Pix, wantImg.(*image.NRGBA).Pix) {
				t.Fatal("reused decoder output differs from package-level decoding")
			}
		})
	}
}

// mustTestSwizzleExpr parses a test channel expression or fails the test.
func mustTestSwizzleExpr(t *testing.T, value string) texconfig.SwizzleExpr {
	t.Helper()
	expr, err := texconfig.ParseSwizzleExpr(value)
	if err != nil {
		t.Fatalf("parse swizzle expression %q: %v", value, err)
	}

	return expr
}

func TestDecodeADSHQFixtureReconstructsOpaqueImage(t *testing.T) {
	data, err := os.ReadFile("testdata/test_adshq.paa")
	if err != nil {
		t.Fatalf("read ADSHQ fixture: %v", err)
	}

	decoded, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode ADSHQ fixture: %v", err)
	}

	var distinctChannels bool
	for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y; y++ {
		for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
			pixel := color.NRGBAModel.Convert(decoded.At(x, y)).(color.NRGBA)
			if pixel.A != 255 || pixel.R != 0 {
				t.Fatalf("decoded ADSHQ pixel at (%d,%d) = %#v, want R=0 and A=255", x, y, pixel)
			}
			if pixel.G != pixel.B {
				distinctChannels = true
			}
		}
	}
	if !distinctChannels {
		t.Fatal("decoded ADSHQ fixture has no distinct G/B channel samples")
	}
}
