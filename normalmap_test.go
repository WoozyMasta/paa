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
