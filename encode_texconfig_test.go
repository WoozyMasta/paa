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
