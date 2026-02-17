// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package texconfig

import "errors"

// Texconfig parse/validation errors. Use errors.Is to check.
var (
	// ErrUnterminatedEscapeInString is returned when a string ends right after '\' escape marker.
	ErrUnterminatedEscapeInString = errors.New("unterminated escape in string")
	// ErrUnterminatedString is returned when a quoted string has no closing quote.
	ErrUnterminatedString = errors.New("unterminated string")
	// ErrUnterminatedBlockComment is returned when a block comment has no closing marker.
	ErrUnterminatedBlockComment = errors.New("unterminated block comment")
	// ErrTextureHintsClassNotFound is returned when TextureHints class is missing in source config.
	ErrTextureHintsClassNotFound = errors.New("TextureHints class not found")
	// ErrInvalidBoolValue is returned when a value cannot be converted to bool.
	ErrInvalidBoolValue = errors.New("invalid bool value")
	// ErrInvalidIntValue is returned when a value cannot be converted to int.
	ErrInvalidIntValue = errors.New("invalid int value")
	// ErrEmptySwizzleExpression is returned when swizzle expression text is empty.
	ErrEmptySwizzleExpression = errors.New("empty swizzle expression")
)
