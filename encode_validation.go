// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"image"
	"reflect"
)

// validateImageBounds checks base dimensions before conversion or pixel access.
func validateImageBounds(img image.Image) (int, int, error) {
	if img == nil {
		return 0, 0, ErrInvalidDimensions
	}

	value := reflect.ValueOf(img)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return 0, 0, ErrInvalidDimensions
		}
	}

	bounds := img.Bounds()
	width, ok := boundsDimension(bounds.Min.X, bounds.Max.X)
	if !ok {
		return 0, 0, ErrInvalidDimensions
	}

	height, ok := boundsDimension(bounds.Min.Y, bounds.Max.Y)
	if !ok {
		return 0, 0, ErrInvalidDimensions
	}

	return width, height, nil
}

// boundsDimension checks a positive rectangle span without overflowing int.
func boundsDimension(lower, upper int) (int, bool) {
	if upper <= lower {
		return 0, false
	}
	if lower < 0 && upper >= 0 && (lower < -0xffff || upper > 0xffff) {
		return 0, false
	}

	span := upper - lower
	if span > 0xffff {
		return 0, false
	}

	return span, true
}

// validateMipDimensions checks dimensions against fields in a PAA mip header.
func validateMipDimensions(paxType PaxType, width, height int) error {
	if width <= 0 || height <= 0 || height > 0xffff {
		return ErrInvalidDimensions
	}

	maxWidth := 0xffff
	if isDXT(paxType) {
		maxWidth = 0x7fff
	}
	if width > maxWidth {
		return ErrInvalidDimensions
	}

	return nil
}

// validateMipPayloadSize checks that a mip fits the 24-bit payload length.
func validateMipPayloadSize(size int) error {
	if size < 0 || size > 0xffffff {
		return ErrMipDataTooLarge
	}

	return nil
}

// validateMipBlocks checks every serialized mip before output is written.
func validateMipBlocks(paxType PaxType, mips []mipBlock) error {
	if len(mips) == 0 {
		return ErrNoMipmaps
	}
	if len(mips) > maxSFFOOffsets {
		return ErrTooManyMipmaps
	}

	for _, mip := range mips {
		if err := validateMipDimensions(paxType, mip.w, mip.h); err != nil {
			return err
		}
		if err := validateMipPayloadSize(len(mip.data)); err != nil {
			return err
		}
	}

	return nil
}
