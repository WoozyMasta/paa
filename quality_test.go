// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/woozymasta/paa/texconfig"
)

// minPSNR is the lowest acceptable PSNR (dB) for a source→PAA→image round-trip.
// Wide-gamut (P3) images with highly saturated colors score ~26 dB on DXT1;
// 24 dB is a generous floor that catches encoder regressions without false
// failures on the deliberately extreme color-gamut test fixtures.
const minPSNR = 24.0

// TestSRGBQuality encodes each fixture image to PAA and back, then measures PSNR.
// Accepts JPEG (.jpg/.jpeg) and PNG (.png) fixtures in testdata/srgb/.
// It runs automatically in go test ./... but is skipped under -short and when
// no fixtures are present (e.g. clean CI checkout).
func TestSRGBQuality(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sRGB quality test in -short mode")
	}

	fixtures := srgbFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no fixtures found in testdata/srgb/ — add JPEG or PNG files to enable this test")
	}

	for _, path := range fixtures {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			orig := mustDecodeImage(t, path)

			var buf bytes.Buffer
			if err := Encode(&buf, orig); err != nil {
				t.Fatalf("Encode: %v", err)
			}

			paaSize := buf.Len()

			decoded, err := Decode(&buf)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}

			p := imagePSNR(orig, decoded)
			t.Logf("PSNR %.2f dB  (size %dx%d, PAA %d B)",
				p, orig.Bounds().Dx(), orig.Bounds().Dy(), paaSize)

			if p < minPSNR {
				t.Errorf("PSNR %.2f dB is below minimum %.2f dB", p, minPSNR)
			}
		})
	}
}

// TestExportSRGBSamples writes PAA files to testdata/srgb/out/
// for manual inspection in Arma/DayZ TexView or similar.
// Skipped unless PAA_EXPORT_SAMPLES=1.
// Suffixes are derived automatically from the default texconfig hints;
// unsupported formats (AI88, ARGB1555, etc.) are logged and skipped.
//
//	PAA_EXPORT_SAMPLES=1 go test -run TestExportSRGBSamples -v
func TestExportSRGBSamples(t *testing.T) {
	if os.Getenv("PAA_EXPORT_SAMPLES") == "" {
		t.Skip("set PAA_EXPORT_SAMPLES=1 to export PAA files to testdata/srgb/out/")
	}

	fixtures := srgbFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no fixtures found in testdata/srgb/")
	}

	cfg, err := texconfig.DefaultTexConvertConfig()
	if err != nil {
		t.Fatalf("texconfig: %v", err)
	}

	// Extract "_suffix" from each hint pattern ("*_co.*" → "_co").
	suffixes := make([]string, 0, len(cfg.Hints))
	for _, h := range cfg.Hints {
		s := strings.TrimPrefix(h.Pattern, "*")
		if idx := strings.LastIndex(s, ".*"); idx >= 0 {
			s = s[:idx]
		}
		suffixes = append(suffixes, s)
	}

	outDir := filepath.Join("testdata", "srgb", "out")
	if err := os.RemoveAll(outDir); err != nil {
		t.Fatalf("clean %s: %v", outDir, err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", outDir, err)
	}

	for _, path := range fixtures {
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		orig := mustDecodeImage(t, path)

		for _, suffix := range suffixes {
			outName := base + suffix + ".paa"
			outPath := filepath.Join(outDir, outName)

			out, err := os.Create(outPath)
			if err != nil {
				t.Fatalf("create %s: %v", outPath, err)
			}

			encErr := EncodeWithFallback(out, orig, outName, "")
			out.Close()

			if encErr != nil {
				os.Remove(outPath)
				t.Logf("skip %s%s: %v", base, suffix, encErr)
				continue
			}

			info, _ := os.Stat(outPath)
			t.Logf("wrote %s (%d B)", outPath, info.Size())
		}
	}
}

// srgbFixtures returns all JPEG and PNG files in testdata/srgb/.
func srgbFixtures(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, pat := range []string{"*.jpg", "*.jpeg", "*.png"} {
		m, _ := filepath.Glob(filepath.Join("testdata", "srgb", pat))
		out = append(out, m...)
	}
	return out
}

// mustDecodeImage opens and decodes an image file, failing the test on error.
func mustDecodeImage(t *testing.T, path string) image.Image {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	return img
}

// imagePSNR computes peak signal-to-noise ratio (dB)
// between two images over the RGB channels, ignoring alpha.
func imagePSNR(a, b image.Image) float64 {
	bounds := a.Bounds()
	var mse float64
	n := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			dr := float64(ar>>8) - float64(br>>8)
			dg := float64(ag>>8) - float64(bg>>8)
			db := float64(ab>>8) - float64(bb>>8)
			mse += dr*dr + dg*dg + db*db
			n++
		}
	}

	if n == 0 {
		return 0
	}

	mse /= float64(n * 3)
	if mse == 0 {
		return math.Inf(1)
	}

	return 10 * math.Log10(255*255/mse)
}
