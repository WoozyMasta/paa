//go:build compat

// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/woozymasta/paa/texconfig"
)

// TestCompatEncode validates our PAA encoder against the official ImageToPAA tool.
// For each texconfig suffix x dimension it:
//  1. Generates a synthetic gradient PNG
//  2. Encodes to PAA with our library
//  3. Runs ImageToPAA to decode the PAA back to PNG
//  4. Checks PSNR for DXT formats without channel swizzle at size ≥ 8x8
//
// PSNR is intentionally not checked for:
//   - Non-DXT formats (AI88, ARGB4444, ARGB1555): lossy palette/grayscale vs RGB source
//   - Swizzled formats (_smdi, _novhq, _detail, …): ImageToPAA emits swizzled channels
//   - 4x4 DXT images: single block yields only 4 interpolated colours (~13 dB floor)
//
// The primary correctness signal is whether ImageToPAA exits 0 and produces output.
//
// Requires IMAGETOPAA_PATH set in env or in a .env file at the project root:
//
//	IMAGETOPAA_PATH=C:\SteamLibrary\steamapps\common\DayZ Tools\Bin\ImageToPAA
//
// Run a subset:
//
//	IMAGETOPAA_PATH=... go test -run 'TestCompatEncode/_ca' -v .
func TestCompatEncode(t *testing.T) {
	toolPath := findImageToPAA(t)

	cfg, err := texconfig.DefaultTexConvertConfig()
	if err != nil {
		t.Fatalf("texconfig: %v", err)
	}

	suffixes := hintSuffixes(cfg)
	dims := testDims()

	// Build suffix → hint so we can skip PSNR where it's not meaningful.
	hintBySuffix := make(map[string]texconfig.TextureHint, len(suffixes))
	for _, suffix := range suffixes {
		if h, ok := texconfig.Resolve("test"+suffix+".paa", cfg); ok {
			hintBySuffix[suffix] = h
		}
	}

	outDir := filepath.Join("testdata", "compat", "encode")
	if err := os.RemoveAll(outDir); err != nil {
		t.Fatalf("clean %s: %v", outDir, err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", outDir, err)
	}

	for _, suffix := range suffixes {
		suffix := suffix
		hint := hintBySuffix[suffix]
		// PSNR is meaningful only for DXT1/DXT5 without channel swizzle and at ≥8x8.
		isDXT := hint.Format == texconfig.TexFormatDXT1 || hint.Format == texconfig.TexFormatDXT5
		noSwizzle := hint.Swizzle.IsIdentity()
		checkPSNR := isDXT && noSwizzle

		t.Run(suffix, func(t *testing.T) {
			for _, d := range dims {
				d := d
				t.Run(fmt.Sprintf("%dx%d", d.w, d.h), func(t *testing.T) {
					src := gradientImage(d.w, d.h)
					name := fmt.Sprintf("test%dx%d%s", d.w, d.h, suffix)
					paaPath := filepath.Join(outDir, name+".paa")
					pngPath := filepath.Join(outDir, name+".png")

					f, err := os.Create(paaPath)
					if err != nil {
						t.Fatalf("create: %v", err)
					}
					encErr := EncodeWithFallback(f, src, name+".paa", "")
					f.Close()
					if encErr != nil {
						_ = os.Remove(paaPath)
						t.Skipf("encode unsupported: %v", encErr)
					}

					cmd := exec.Command(toolPath, paaPath, pngPath) //nolint:gosec
					if out, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("ImageToPAA decode failed: %v\n%s", err, out)
					}
					if _, err := os.Stat(pngPath); err != nil {
						t.Fatalf("ImageToPAA produced no PNG: %v", err)
					}

					// PSNR check: DXT without swizzle at ≥8x8 only.
					// 4x4 DXT has a single block → ~13 dB floor (expected, not a bug).
					if !checkPSNR || d.w < 8 || d.h < 8 {
						t.Logf("PSNR check skipped (format=%v swizzle=%v %dx%d)", hint.Format, !noSwizzle, d.w, d.h)
						return
					}

					decoded := mustDecodeImage(t, pngPath)
					p := imagePSNR(src, decoded)
					t.Logf("PSNR=%.2f dB (%dx%d)", p, d.w, d.h)
					if p < 20.0 {
						t.Errorf("PSNR %.2f dB below 20 dB floor", p)
					}
				})
			}
		})
	}
}

// TestCompatDecode validates our PAA decoder against the official ImageToPAA tool.
// For each texconfig suffix x dimension it:
//  1. Generates a synthetic gradient PNG with a suffix-bearing name
//  2. Runs ImageToPAA to encode it to PAA (suffix drives format selection)
//  3. Decodes the PAA with our library and verifies dimensions and no error
//
// PSNR is not checked here: the source image passes through ImageToPAA's encoder
// (which applies format-specific preprocessing: alpha noise, mipmap fade-out,
// normal-map normalization, swizzles, etc.) so comparing decoded pixels against
// the original gradient would measure ImageToPAA's encoder quality, not ours.
// The correctness signal is: our decoder does not error and returns correct dimensions.
//
// Same IMAGETOPAA_PATH requirement as TestCompatEncode.
func TestCompatDecode(t *testing.T) {
	toolPath := findImageToPAA(t)

	cfg, err := texconfig.DefaultTexConvertConfig()
	if err != nil {
		t.Fatalf("texconfig: %v", err)
	}

	suffixes := hintSuffixes(cfg)
	dims := testDims()

	outDir := filepath.Join("testdata", "compat", "decode")
	if err := os.RemoveAll(outDir); err != nil {
		t.Fatalf("clean %s: %v", outDir, err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", outDir, err)
	}

	for _, suffix := range suffixes {
		suffix := suffix
		t.Run(suffix, func(t *testing.T) {
			for _, d := range dims {
				d := d
				t.Run(fmt.Sprintf("%dx%d", d.w, d.h), func(t *testing.T) {
					src := gradientImage(d.w, d.h)
					name := fmt.Sprintf("test%dx%d%s", d.w, d.h, suffix)
					pngPath := filepath.Join(outDir, name+".png")
					paaPath := filepath.Join(outDir, name+".paa")

					writeTestPNG(t, pngPath, src)

					// Single-arg form: ImageToPAA derives .paa path from .png path.
					cmd := exec.Command(toolPath, pngPath) //nolint:gosec
					if out, err := cmd.CombinedOutput(); err != nil {
						t.Skipf("ImageToPAA encode skipped: %v\n%s", err, out)
					}
					if _, err := os.Stat(paaPath); err != nil {
						t.Skipf("ImageToPAA produced no PAA: %v", err)
					}

					paaFile, err := os.Open(paaPath)
					if err != nil {
						t.Fatalf("open PAA: %v", err)
					}
					decoded, decErr := Decode(paaFile)
					paaFile.Close()
					if decErr != nil {
						t.Fatalf("our Decode: %v", decErr)
					}

					got := decoded.Bounds()
					if got.Dx() != d.w || got.Dy() != d.h {
						t.Errorf("decoded size %dx%d, want %dx%d", got.Dx(), got.Dy(), d.w, d.h)
					}
					t.Logf("ok %dx%d", got.Dx(), got.Dy())
				})
			}
		})
	}
}

// findImageToPAA resolves the ImageToPAA executable path from env or .env file.
// Skips the test if neither is configured or the binary does not exist.
func findImageToPAA(t *testing.T) string {
	t.Helper()

	path := os.Getenv("IMAGETOPAA_PATH")
	if path == "" {
		if env := parseDotEnv(filepath.Join(".", ".env")); env != nil {
			path = env["IMAGETOPAA_PATH"]
		}
	}
	if path == "" {
		t.Skip("set IMAGETOPAA_PATH in env or .env to enable compat tests")
	}

	// Accept either a directory (containing ImageToPAA.exe) or the exe itself.
	exe := filepath.Join(path, "ImageToPAA.exe")
	if _, err := os.Stat(exe); err != nil {
		exe = path
		if _, err := os.Stat(exe); err != nil {
			t.Skipf("ImageToPAA not found at %q", path)
		}
	}

	return exe
}

// parseDotEnv reads KEY=VALUE pairs from a file, ignoring blank lines and # comments.
func parseDotEnv(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	m := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return m
}

// hintSuffixes extracts the filename suffix from each texconfig hint pattern.
// "*_co.*" → "_co".
func hintSuffixes(cfg texconfig.TexConvertConfig) []string {
	seen := make(map[string]bool, len(cfg.Hints))
	suffixes := make([]string, 0, len(cfg.Hints))
	for _, h := range cfg.Hints {
		s := strings.TrimPrefix(h.Pattern, "*")
		if idx := strings.LastIndex(s, ".*"); idx >= 0 {
			s = s[:idx]
		}
		if !seen[s] {
			seen[s] = true
			suffixes = append(suffixes, s)
		}
	}
	return suffixes
}

// testDims returns the PoT dimension matrix used for compat testing.
// Covers 1:1, 2:1, 4:1, 8:1, 16:1 ratios and their portrait inverses.
func testDims() []struct{ w, h int } {
	return []struct{ w, h int }{
		// 1:1 squares
		{4, 4}, {8, 8}, {16, 16}, {32, 32}, {64, 64}, {128, 128}, {256, 256}, {512, 512},
		// 2:1 landscape / 1:2 portrait
		{8, 4}, {16, 8}, {32, 16}, {64, 32}, {128, 64}, {256, 128}, {512, 256},
		{4, 8}, {8, 16}, {16, 32}, {32, 64}, {64, 128}, {128, 256}, {256, 512},
		// 4:1 / 1:4
		{16, 4}, {32, 8}, {64, 16}, {128, 32}, {256, 64}, {512, 128},
		{4, 16}, {8, 32}, {16, 64}, {32, 128}, {64, 256}, {128, 512},
		// 8:1 / 1:8
		{32, 4}, {64, 8}, {128, 16}, {256, 32}, {512, 64},
		{4, 32}, {8, 64}, {16, 128}, {32, 256}, {64, 512},
		// 16:1 / 1:16
		{64, 4}, {128, 8}, {256, 16}, {512, 32},
		{4, 64}, {8, 128}, {16, 256}, {32, 512},
	}
}

// gradientImage generates a synthetic NRGBA gradient for round-trip quality tests.
// R varies along X, G along Y, B is mixed, A is fully opaque.
func gradientImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	wm, hm := max(w-1, 1), max(h-1, 1)
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 255 / wm),
				G: uint8(y * 255 / hm),
				B: uint8((x + y) * 127 / (wm + hm)),
				A: 255,
			})
		}
	}
	return img
}

// writeTestPNG encodes img to a PNG file at path, failing the test on error.
func writeTestPNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create PNG %s: %v", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode PNG %s: %v", path, err)
	}
}
