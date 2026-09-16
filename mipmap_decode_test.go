// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"errors"
	"image"
	"testing"
)

func TestMipMapImageRequiresExactPayloadSize(t *testing.T) {
	tests := []struct {
		name   string
		format PaxType
		size   int
	}{
		{name: "ARGB8", format: PaxARGB8, size: 16},
		{name: "ARGBA5", format: PaxARGBA5, size: 8},
		{name: "ARGB4", format: PaxARGB4, size: 8},
		{name: "GRAYA", format: PaxGRAYA, size: 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := make([]byte, tt.size)
			mip := &MipMap{Data: payload, Type: tt.format, Width: 2, Height: 2}
			if _, err := mip.Image(); err != nil {
				t.Fatalf("Image() with exact payload: %v", err)
			}

			mip.Data = payload[:len(payload)-1]
			if img, err := mip.Image(); !errors.Is(err, ErrInsufficientData) || img != nil {
				t.Fatalf("Image() with short payload = (%v, %v), want (nil, ErrInsufficientData)", img, err)
			}

			mip.Data = append(payload, 0)
			if img, err := mip.Image(); !errors.Is(err, ErrInsufficientData) || img != nil {
				t.Fatalf("Image() with oversized payload = (%v, %v), want (nil, ErrInsufficientData)", img, err)
			}
		})
	}
}

func TestDecodePixelFormatRequiresExactPayloadSize(t *testing.T) {
	tests := []struct {
		name   string
		format PaxType
		size   int
	}{
		{name: "ARGB8", format: PaxARGB8, size: 16},
		{name: "ARGBA5", format: PaxARGBA5, size: 8},
		{name: "ARGB4", format: PaxARGB4, size: 8},
		{name: "GRAYA", format: PaxGRAYA, size: 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
			payload := make([]byte, tt.size)
			if err := decodePixelFormat(tt.format, payload, 2, 2, img); err != nil {
				t.Fatalf("decodePixelFormat() with exact payload: %v", err)
			}

			for _, malformed := range [][]byte{payload[:len(payload)-1], append(payload, 0)} {
				if err := decodePixelFormat(tt.format, malformed, 2, 2, img); !errors.Is(err, ErrInsufficientData) {
					t.Fatalf("decodePixelFormat() with %d bytes = %v, want ErrInsufficientData", len(malformed), err)
				}
			}
		})
	}
}

func TestMipMapImageValidatesReceiverDimensionsAndBudget(t *testing.T) {
	var nilMip *MipMap
	if _, err := nilMip.ImageWithOptions(nil); !errors.Is(err, ErrNilMipMap) {
		t.Fatalf("nil mip ImageWithOptions() error = %v, want ErrNilMipMap", err)
	}

	for _, tt := range []struct {
		name string
		mip  *MipMap
		want error
	}{
		{
			name: "zero dimension",
			mip:  &MipMap{Type: PaxARGB8, Width: 0, Height: 1},
			want: ErrInvalidDimensions,
		},
		{
			name: "over budget",
			mip:  &MipMap{Type: PaxARGB8, Width: 0xFFFF, Height: 0xFFFF},
			want: ErrImageTooLarge,
		},
		{
			name: "unsupported format",
			mip:  &MipMap{Type: PaxType(0xFFFF), Width: 1, Height: 1},
			want: ErrUnsupportedPixelFmt,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if img, err := tt.mip.Image(); !errors.Is(err, tt.want) || img != nil {
				t.Fatalf("Image() = (%v, %v), want (nil, %v)", img, err, tt.want)
			}
		})
	}
}
