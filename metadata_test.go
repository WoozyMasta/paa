package paa

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeMetadata_ParityWithDecodePAA_Fixtures(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping fixture parity test in short mode")
	}

	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := strings.ToLower(e.Name())
		if strings.HasPrefix(name, "test_") && strings.HasSuffix(name, ".paa") {
			files = append(files, filepath.Join("testdata", e.Name()))
		}
	}

	if len(files) == 0 {
		t.Fatalf("no testdata/test_*.paa files found")
	}

	for _, path := range files {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			meta, err := DecodeMetadata(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("DecodeMetadata: %v", err)
			}
			headers, err := DecodeMetadataHeaders(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("DecodeMetadataHeaders: %v", err)
			}

			full, err := DecodePAA(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("DecodePAA: %v", err)
			}

			if meta.Type != full.Type {
				t.Fatalf("type mismatch: meta=%d full=%d", meta.Type, full.Type)
			}

			if len(meta.MipHeaders) != len(full.MipMaps) {
				t.Fatalf("mipmap count mismatch: meta=%d full=%d", len(meta.MipHeaders), len(full.MipMaps))
			}
			if headers.Type != meta.Type {
				t.Fatalf("headers type mismatch: headers=%d meta=%d", headers.Type, meta.Type)
			}
			if len(headers.MipHeaders) != len(meta.MipHeaders) {
				t.Fatalf("headers mipmap count mismatch: headers=%d meta=%d", len(headers.MipHeaders), len(meta.MipHeaders))
			}

			offsets := nonZeroSFFOOffsets(meta.Taggs["SFFO"])
			if len(offsets) != len(meta.MipHeaders) {
				t.Fatalf("SFFO non-zero offsets=%d mip headers=%d", len(offsets), len(meta.MipHeaders))
			}

			for i := range meta.MipHeaders {
				if meta.MipHeaders[i].Offset != offsets[i] {
					t.Fatalf("offset[%d] mismatch: meta=%d sffo=%d", i, meta.MipHeaders[i].Offset, offsets[i])
				}

				if meta.MipHeaders[i].Width != full.MipMaps[i].Width {
					t.Fatalf("width[%d] mismatch: meta=%d full=%d", i, meta.MipHeaders[i].Width, full.MipMaps[i].Width)
				}

				if meta.MipHeaders[i].Height != full.MipMaps[i].Height {
					t.Fatalf("height[%d] mismatch: meta=%d full=%d", i, meta.MipHeaders[i].Height, full.MipMaps[i].Height)
				}
				if headers.MipHeaders[i] != meta.MipHeaders[i] {
					t.Fatalf("headers mip[%d] mismatch: headers=%+v meta=%+v", i, headers.MipHeaders[i], meta.MipHeaders[i])
				}
			}

			cgva := meta.Taggs["CGVA"]
			if len(cgva) >= 4 {
				if !headers.HasAverageColor || !bytes.Equal(headers.AverageColor[:], cgva[:4]) {
					t.Fatalf("headers CGVA mismatch: has=%v headers=%v meta=%v", headers.HasAverageColor, headers.AverageColor, cgva[:4])
				}
			} else if headers.HasAverageColor {
				t.Fatalf("headers unexpectedly has CGVA")
			}

			cxam := meta.Taggs["CXAM"]
			if len(cxam) >= 4 {
				if !headers.HasMaxColor || !bytes.Equal(headers.MaxColor[:], cxam[:4]) {
					t.Fatalf("headers CXAM mismatch: has=%v headers=%v meta=%v", headers.HasMaxColor, headers.MaxColor, cxam[:4])
				}
			} else if headers.HasMaxColor {
				t.Fatalf("headers unexpectedly has CXAM")
			}

			galf := meta.Taggs["GALF"]
			if len(galf) >= 4 {
				wantGALF := binary.LittleEndian.Uint32(galf[:4])
				if !headers.HasGALF || headers.GALF != wantGALF {
					t.Fatalf("headers GALF mismatch: has=%v got=%d want=%d", headers.HasGALF, headers.GALF, wantGALF)
				}
			} else if headers.HasGALF {
				t.Fatalf("headers unexpectedly has GALF")
			}
		})
	}
}

func TestDecodeMetadataBytesHelpers(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 17), G: uint8(y * 19), B: 123, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, &EncodeOptions{Type: PaxDXT1}); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}
	raw := buf.Bytes()

	metaFromReader, err := DecodeMetadata(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeMetadata(reader): %v", err)
	}
	metaFromBytes, err := DecodeMetadataBytes(raw)
	if err != nil {
		t.Fatalf("DecodeMetadataBytes: %v", err)
	}
	if metaFromReader.Type != metaFromBytes.Type || len(metaFromReader.MipHeaders) != len(metaFromBytes.MipHeaders) {
		t.Fatalf("metadata mismatch: reader=%+v bytes=%+v", metaFromReader.Type, metaFromBytes.Type)
	}

	headersFromReader, err := DecodeMetadataHeaders(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeMetadataHeaders(reader): %v", err)
	}
	headersFromBytes, err := DecodeMetadataHeadersBytes(raw)
	if err != nil {
		t.Fatalf("DecodeMetadataHeadersBytes: %v", err)
	}
	if headersFromReader.Type != headersFromBytes.Type || len(headersFromReader.MipHeaders) != len(headersFromBytes.MipHeaders) {
		t.Fatalf("metadata headers mismatch: reader=%+v bytes=%+v", headersFromReader.Type, headersFromBytes.Type)
	}
}

// nonZeroSFFOOffsets returns only non-zero 32-bit entries from SFFO table.
func nonZeroSFFOOffsets(sffo []byte) []uint32 {
	out := make([]uint32, 0, len(sffo)/4)
	for i := 0; i+4 <= len(sffo); i += 4 {
		v := binary.LittleEndian.Uint32(sffo[i : i+4])
		if v == 0 {
			continue
		}

		out = append(out, v)
	}

	return out
}
