// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"image"

	"github.com/woozymasta/paa/texconfig"
)

// copyNRGBAInto copies source pixels into a reusable tight origin-zero image.
func copyNRGBAInto(dst, src *image.NRGBA) *image.NRGBA {
	bounds := src.Bounds()
	dst = reuseNRGBA(dst, bounds.Dx(), bounds.Dy())
	for y := 0; y < bounds.Dy(); y++ {
		srcRow := y * src.Stride
		dstRow := y * dst.Stride
		copy(dst.Pix[dstRow:dstRow+bounds.Dx()*4], src.Pix[srcRow:srcRow+bounds.Dx()*4])
	}

	return dst
}

// applyChannelSwizzleInPlace transforms a tight NRGBA image without a temp image.
func applyChannelSwizzleInPlace(img *image.NRGBA, swz texconfig.ChannelSwizzle) {
	for y := 0; y < img.Rect.Dy(); y++ {
		row := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			off := row + x*4
			r, g, b, a := img.Pix[off], img.Pix[off+1], img.Pix[off+2], img.Pix[off+3]
			img.Pix[off+0] = applyEncodeSwizzleExpr(swz.R, texconfig.SwizzleR, r, g, b, a)
			img.Pix[off+1] = applyEncodeSwizzleExpr(swz.G, texconfig.SwizzleG, r, g, b, a)
			img.Pix[off+2] = applyEncodeSwizzleExpr(swz.B, texconfig.SwizzleB, r, g, b, a)
			img.Pix[off+3] = applyEncodeSwizzleExpr(swz.A, texconfig.SwizzleA, r, g, b, a)
		}
	}
}

// applyEncodeSwizzleExpr evaluates a TexConfig expression over cached source channels.
func applyEncodeSwizzleExpr(expr texconfig.SwizzleExpr, fallback texconfig.SwizzleSource, r, g, b, a byte) byte {
	if !expr.Valid {
		return encodeSwizzleChannel(fallback, r, g, b, a)
	}
	if expr.IsConst {
		return expr.ConstValue
	}

	value := encodeSwizzleChannel(expr.Source, r, g, b, a)
	if expr.Invert {
		return 255 - value
	}

	return value
}

// encodeSwizzleChannel returns one channel selected by a TexConfig expression.
func encodeSwizzleChannel(source texconfig.SwizzleSource, r, g, b, a byte) byte {
	switch source {
	case texconfig.SwizzleA:
		return a
	case texconfig.SwizzleR:
		return r
	case texconfig.SwizzleG:
		return g
	case texconfig.SwizzleB:
		return b
	default:
		return 0
	}
}
