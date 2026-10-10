// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package texconfig

import (
	"image"
	"image/color"
	"testing"
)

// TestApplyChannelSwizzlePreservesNonZeroOrigin checks public helper bounds handling.
func TestApplyChannelSwizzlePreservesNonZeroOrigin(t *testing.T) {
	src := image.NewNRGBA(image.Rect(7, 11, 8, 12))
	src.SetNRGBA(7, 11, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
	swz := ChannelSwizzle{
		R: mustSwizzleExprForTest(t, "G"), G: mustSwizzleExprForTest(t, "B"),
		B: mustSwizzleExprForTest(t, "A"), A: mustSwizzleExprForTest(t, "R"),
	}

	got := ApplyChannelSwizzle(src, swz)
	if got.Bounds() != src.Bounds() {
		t.Fatalf("bounds = %v, want %v", got.Bounds(), src.Bounds())
	}
	if pixel := got.NRGBAAt(7, 11); pixel != (color.NRGBA{R: 20, G: 30, B: 40, A: 10}) {
		t.Fatalf("pixel = %#v, want %#v", pixel, color.NRGBA{R: 20, G: 30, B: 40, A: 10})
	}
}

// mustSwizzleExprForTest parses an expression or fails the current test.
func mustSwizzleExprForTest(t *testing.T, value string) SwizzleExpr {
	t.Helper()
	expr, err := ParseSwizzleExpr(value)
	if err != nil {
		t.Fatalf("parse swizzle expression %q: %v", value, err)
	}

	return expr
}
