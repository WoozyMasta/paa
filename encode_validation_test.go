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
)

type boundsOnlyImage struct {
	bounds image.Rectangle
}

func (img boundsOnlyImage) ColorModel() color.Model {
	return color.NRGBAModel
}

func (img boundsOnlyImage) Bounds() image.Rectangle {
	return img.bounds
}

func (boundsOnlyImage) At(int, int) color.Color {
	return color.NRGBA{A: 255}
}

func TestValidateMipDimensions(t *testing.T) {
	tests := []struct {
		name    string
		paxType PaxType
		width   int
		height  int
		wantErr error
	}{
		{name: "DXT maximum width", paxType: PaxDXT1, width: 0x7fff, height: 1},
		{name: "DXT reserved width bit", paxType: PaxDXT1, width: 0x8000, height: 1, wantErr: ErrInvalidDimensions},
		{name: "non-DXT maximum width", paxType: PaxARGB8, width: 0xffff, height: 1},
		{name: "height exceeds uint16", paxType: PaxARGB8, width: 1, height: 0x10000, wantErr: ErrInvalidDimensions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateMipDimensions(tt.paxType, tt.width, tt.height); !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateMipDimensions() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMipBlocksDXTWidthIgnoresLZOFlag(t *testing.T) {
	for _, useLZ := range []bool{false, true} {
		mips := []mipBlock{{w: 0x8000, h: 1, useLZ: useLZ}}
		if err := validateMipBlocks(PaxDXT1, mips); !errors.Is(err, ErrInvalidDimensions) {
			t.Errorf("validateMipBlocks(useLZ=%t) error = %v, want ErrInvalidDimensions", useLZ, err)
		}
	}
}

func TestValidateMipPayloadSize(t *testing.T) {
	if err := validateMipPayloadSize(0xffffff); err != nil {
		t.Fatalf("validateMipPayloadSize(max) error = %v", err)
	}
	if err := validateMipPayloadSize(0x1000000); !errors.Is(err, ErrMipDataTooLarge) {
		t.Fatalf("validateMipPayloadSize(max+1) error = %v, want ErrMipDataTooLarge", err)
	}
}

func TestEncodePreflightWritesNoBytesForInvalidDimensions(t *testing.T) {
	tests := []struct {
		name string
		img  image.Image
		opts *EncodeOptions
	}{
		{name: "explicit DXT", img: boundsOnlyImage{image.Rect(0, 0, 0x8000, 1)}, opts: &EncodeOptions{Type: PaxDXT1}},
		{name: "automatic DXT", img: boundsOnlyImage{image.Rect(0, 0, 0x8000, 1)}},
		{name: "empty bounds", img: boundsOnlyImage{image.Rectangle{}}},
		{name: "typed nil", img: (*image.NRGBA)(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := EncodeWithOptions(&out, tt.img, tt.opts); !errors.Is(err, ErrInvalidDimensions) {
				t.Fatalf("EncodeWithOptions() error = %v, want ErrInvalidDimensions", err)
			}
			if out.Len() != 0 {
				t.Fatalf("EncodeWithOptions() wrote %d bytes before preflight error", out.Len())
			}
		})
	}
}

func TestEncodeFromCompressedBlocksRejectsTooManyMipmapsBeforeWriting(t *testing.T) {
	mipmaps := make([][]byte, maxSFFOOffsets+1)
	for i := range mipmaps {
		mipmaps[i] = make([]byte, 8)
	}

	tests := []struct {
		name string
		call func(*bytes.Buffer) error
	}{
		{
			name: "DDS",
			call: func(out *bytes.Buffer) error {
				dds := &bcn.DDS{Format: bcn.FormatDXT1, Width: 1, Height: 1, Faces: []bcn.Face{{Mipmaps: mipmaps}}}
				return EncodeFromDDS(out, dds, nil)
			},
		},
		{
			name: "KTX",
			call: func(out *bytes.Buffer) error {
				ktx := &bcn.KTX{Format: bcn.FormatDXT1, Width: 1, Height: 1, Faces: []bcn.Face{{Mipmaps: mipmaps}}}
				return EncodeFromKTX(out, ktx, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := tt.call(&out); !errors.Is(err, ErrTooManyMipmaps) {
				t.Fatalf("encode() error = %v, want ErrTooManyMipmaps", err)
			}
			if out.Len() != 0 {
				t.Fatalf("encode() wrote %d bytes before mip count error", out.Len())
			}
		})
	}
}
