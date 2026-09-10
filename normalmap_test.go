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
)

func TestSwizzleNormalMapPreservesStraightAlpha(t *testing.T) {
	src := image.NewNRGBA(image.Rect(7, 11, 8, 12))
	src.SetNRGBA(7, 11, color.NRGBA{R: 128, G: 128, B: 255, A: 255})

	swizzled := swizzleNormalMap(src)
	got, ok := swizzled.(*image.NRGBA)
	if !ok {
		t.Fatalf("swizzled image type = %T, want *image.NRGBA", swizzled)
	}
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
