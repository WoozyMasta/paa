// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"errors"
	"testing"

	"github.com/woozymasta/bcn"
	"github.com/woozymasta/lzo"
)

func TestEncodeFromContainersRejectsInvalidMipSize(t *testing.T) {
	mipmaps := [][]byte{make([]byte, 32), make([]byte, 7)}
	tests := []struct {
		name string
		call func(*bytes.Buffer) error
	}{
		{
			name: "DDS",
			call: func(out *bytes.Buffer) error {
				dds := &bcn.DDS{Format: bcn.FormatDXT1, Width: 8, Height: 8, Faces: []bcn.Face{{Mipmaps: mipmaps}}}
				return EncodeFromDDS(out, dds, nil)
			},
		},
		{
			name: "KTX",
			call: func(out *bytes.Buffer) error {
				ktx := &bcn.KTX{Format: bcn.FormatDXT1, Width: 8, Height: 8, Faces: []bcn.Face{{Mipmaps: mipmaps}}}
				return EncodeFromKTX(out, ktx, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := tt.call(&out); !errors.Is(err, ErrInsufficientData) {
				t.Fatalf("encode error = %v, want ErrInsufficientData", err)
			}
			if out.Len() != 0 {
				t.Fatalf("encode wrote %d bytes before rejecting mip size", out.Len())
			}
		})
	}
}

func TestDirectExportRejectsNonCanonicalMipDimensions(t *testing.T) {
	p := &PAA{
		Type: PaxDXT1,
		MipMaps: []*MipMap{
			{Type: PaxDXT1, Width: 8, Height: 8, Data: make([]byte, 32)},
			{Type: PaxDXT1, Width: 3, Height: 4, Data: make([]byte, 8)},
		},
	}

	if _, err := p.ToDDS(); !errors.Is(err, ErrInvalidDimensions) {
		t.Fatalf("ToDDS error = %v, want ErrInvalidDimensions", err)
	}
	if _, err := p.ToKTX(); !errors.Is(err, ErrInvalidDimensions) {
		t.Fatalf("ToKTX error = %v, want ErrInvalidDimensions", err)
	}
}

func TestCompressDirectMipPropagatesLZOFailure(t *testing.T) {
	wantErr := errors.New("injected LZO failure")
	called := false
	compressed, useLZO, err := compressDirectMip(
		make([]byte, 16),
		&EncodeOptions{LZOLevel: 9},
		func(src, dst []byte, opts *lzo.CompressOptions) ([]byte, error) {
			called = true
			if opts == nil || opts.Level != 9 {
				t.Fatalf("LZO options = %v, want level 9", opts)
			}
			return nil, wantErr
		},
	)

	if !called {
		t.Fatal("LZO compressor was not called")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("compression error = %v, want injected LZO failure", err)
	}
	if compressed != nil || useLZO {
		t.Fatalf("compression result = (%d bytes, useLZO=%t), want no result", len(compressed), useLZO)
	}
}

func TestDirectDDSKTXRoundTripCanonicalDXTMipChains(t *testing.T) {
	tests := []struct {
		name   string
		pax    PaxType
		format bcn.Format
	}{
		{name: "DXT1", pax: PaxDXT1, format: bcn.FormatDXT1},
		{name: "DXT3", pax: PaxDXT3, format: bcn.FormatDXT3},
		{name: "DXT5", pax: PaxDXT5, format: bcn.FormatDXT5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mipmaps := make([][]byte, 0, 4)
			width, height := 8, 8
			for range 4 {
				mipmaps = append(mipmaps, bytes.Repeat([]byte{byte(width + height)}, expectedMipSize(tt.pax, width, height)))
				width = max(1, width/2)
				height = max(1, height/2)
			}

			for _, container := range []string{"DDS", "KTX"} {
				t.Run(container, func(t *testing.T) {
					var encoded bytes.Buffer
					switch container {
					case "DDS":
						in := &bcn.DDS{Format: tt.format, Width: 8, Height: 8, Faces: []bcn.Face{{Mipmaps: mipmaps}}}
						if err := EncodeFromDDS(&encoded, in, nil); err != nil {
							t.Fatalf("EncodeFromDDS: %v", err)
						}

					case "KTX":
						in := &bcn.KTX{Format: tt.format, Width: 8, Height: 8, Faces: []bcn.Face{{Mipmaps: mipmaps}}}
						if err := EncodeFromKTX(&encoded, in, nil); err != nil {
							t.Fatalf("EncodeFromKTX: %v", err)
						}
					}

					paa, err := DecodePAA(bytes.NewReader(encoded.Bytes()))
					if err != nil {
						t.Fatalf("DecodePAA: %v", err)
					}
					if len(paa.MipMaps) != len(mipmaps) {
						t.Fatalf("decoded mip count = %d, want %d", len(paa.MipMaps), len(mipmaps))
					}

					var exported [][]byte
					if container == "DDS" {
						out, err := paa.ToDDS()
						if err != nil {
							t.Fatalf("ToDDS: %v", err)
						}
						exported = out.Faces[0].Mipmaps
					} else {
						out, err := paa.ToKTX()
						if err != nil {
							t.Fatalf("ToKTX: %v", err)
						}
						exported = out.Faces[0].Mipmaps
					}

					for i := range mipmaps {
						if !bytes.Equal(exported[i], mipmaps[i]) {
							t.Fatalf("mipmap[%d] payload changed in direct round-trip", i)
						}
					}
				})
			}
		})
	}
}
