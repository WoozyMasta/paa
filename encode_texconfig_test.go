// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/woozymasta/bcn"
	"github.com/woozymasta/paa/texconfig"
)

func TestEncodeWithTexConfigFormats(t *testing.T) {
	tests := []struct {
		name      string
		format    texconfig.TexFormat
		alpha     uint8
		want      PaxType
		wantError error
	}{
		{name: "default opaque", format: texconfig.TexFormatDefault, alpha: 255, want: PaxDXT1},
		{name: "default alpha", format: texconfig.TexFormatDefault, alpha: 128, want: PaxDXT5},
		{name: "DXT1", format: texconfig.TexFormatDXT1, alpha: 255, want: PaxDXT1},
		{name: "DXT3", format: texconfig.TexFormatDXT3, alpha: 255, want: PaxDXT3},
		{name: "DXT5", format: texconfig.TexFormatDXT5, alpha: 255, want: PaxDXT5},
		{name: "DXT2", format: texconfig.TexFormatDXT2, alpha: 255, wantError: ErrUnsupportedFormat},
		{name: "DXT4", format: texconfig.TexFormatDXT4, alpha: 255, wantError: ErrUnsupportedFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := texconfig.DefaultTexConvertConfig()
			if err != nil {
				t.Fatalf("DefaultTexConvertConfig: %v", err)
			}
			cfg.DisableLZO = true
			found := false
			for i := range cfg.Hints {
				if cfg.Hints[i].Pattern == "*_co.*" {
					cfg.Hints[i].Format = tt.format
					found = true
					break
				}
			}
			if !found {
				t.Fatal("default config has no color-map hint")
			}

			img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					img.SetNRGBA(x, y, color.NRGBA{R: 180, G: 100, B: 40, A: tt.alpha})
				}
			}

			var encoded bytes.Buffer
			err = EncodeWithTexConfig(&encoded, img, "texture_co.paa", cfg)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("EncodeWithTexConfig error = %v, want %v", err, tt.wantError)
				}
				if encoded.Len() != 0 {
					t.Fatalf("EncodeWithTexConfig wrote %d bytes before rejecting format", encoded.Len())
				}
				return
			}
			if err != nil {
				t.Fatalf("EncodeWithTexConfig: %v", err)
			}

			decoded, err := DecodePAA(bytes.NewReader(encoded.Bytes()))
			if err != nil {
				t.Fatalf("DecodePAA: %v", err)
			}
			if decoded.Type != tt.want {
				t.Fatalf("PAA type = %v, want %v", decoded.Type, tt.want)
			}
			if _, err := decoded.MipMaps[0].Image(); err != nil {
				t.Fatalf("decode first mip: %v", err)
			}
		})
	}
}

func TestAutoReduceIfNeededMatchesFullMipChain(t *testing.T) {
	for _, tt := range []struct {
		name   string
		width  int
		height int
		limit  int
	}{
		{name: "power of two", width: 16, height: 8, limit: 4},
		{name: "odd dimensions", width: 9, height: 7, limit: 4},
		{name: "odd dimensions multiple levels", width: 13, height: 9, limit: 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewNRGBA(image.Rect(0, 0, tt.width, tt.height))
			for i := range img.Pix {
				img.Pix[i] = byte(i*37 + 11)
			}
			autoReduce := true
			hint := texconfig.TextureHint{AutoReduce: &autoReduce, LimitSize: tt.limit}
			got := autoReduceIfNeeded(img, hint, texconfig.TexConvertConfig{})

			var want image.Image = img
			for _, mip := range bcn.GenerateMipmaps(img, false) {
				if mip.Bounds().Dx() <= tt.limit && mip.Bounds().Dy() <= tt.limit {
					want = mip
					break
				}
			}
			if got.Bounds() != want.Bounds() {
				t.Fatalf("reduced bounds = %v, want %v", got.Bounds(), want.Bounds())
			}
			if !bytes.Equal(got.(*image.NRGBA).Pix, want.(*image.NRGBA).Pix) {
				t.Fatal("reduced pixels differ from the matching full-chain mip")
			}
		})
	}
}

func TestAutoReduceIfNeededNoOpCases(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	autoReduce := true
	noAutoReduce := false
	for _, tt := range []struct {
		name string
		hint texconfig.TextureHint
		cfg  texconfig.TexConvertConfig
	}{
		{
			name: "within limit",
			hint: texconfig.TextureHint{AutoReduce: &autoReduce, LimitSize: 16},
		},
		{
			name: "non-positive limit",
			hint: texconfig.TextureHint{AutoReduce: &autoReduce, LimitSize: 0},
		},
		{
			name: "missing hint",
			hint: texconfig.TextureHint{LimitSize: 4},
		},
		{
			name: "disabled hint",
			hint: texconfig.TextureHint{AutoReduce: &noAutoReduce, LimitSize: 4},
		},
		{
			name: "disabled config",
			hint: texconfig.TextureHint{AutoReduce: &autoReduce, LimitSize: 4},
			cfg:  texconfig.TexConvertConfig{DisableAutoReduce: true},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := autoReduceIfNeeded(img, tt.hint, tt.cfg); got != img {
				t.Fatal("autoReduceIfNeeded changed image for a no-op case")
			}
		})
	}
}

func TestAutoReduceIfNeededPreservesSRGBSelection(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			v := uint8(0)
			if (x+y)%2 != 0 {
				v = 255
			}
			o := img.PixOffset(x, y)
			img.Pix[o+0], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = v, v, v, 255
		}
	}

	autoReduce, dynRange := true, true
	hint := texconfig.TextureHint{
		AutoReduce: &autoReduce,
		DynRange:   &dynRange,
		LimitSize:  4,
	}
	got := autoReduceIfNeeded(img, hint, texconfig.TexConvertConfig{UseSRGBFromDynRange: true})
	want := bcn.GenerateMipmapsN(img, 2, true)[1]
	if !bytes.Equal(got.(*image.NRGBA).Pix, want.Pix) {
		t.Fatal("sRGB-reduced pixels differ from the expected mip")
	}
}

func TestScanAlphaNRGBAFastPathMatchesImageInterface(t *testing.T) {
	base := image.NewNRGBA(image.Rect(3, 5, 11, 12))
	img := base.SubImage(image.Rect(5, 6, 10, 11)).(*image.NRGBA)
	wrapped := struct{ image.Image }{Image: img}

	for _, alpha := range []uint8{255, 0, 239, 240} {
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				offset := img.PixOffset(x, y)
				img.Pix[offset+3] = alpha
			}
		}

		if got, want := scanAlpha(img), scanAlpha(wrapped); got != want {
			t.Fatalf("scanAlpha with alpha %d = %+v, want %+v", alpha, got, want)
		}
	}
}
