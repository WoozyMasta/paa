package paa

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeMetadata_ParityWithDecodePAA_Fixtures(t *testing.T) {
	t.Parallel()

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
			}
		})
	}
}

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
