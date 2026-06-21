// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package texconfig

import (
	"path/filepath"
	"strings"
)

// TexConvertResolver resolves texture hints by filename.
type TexConvertResolver struct {
	config TexConvertConfig
}

// NewTexConvertResolver creates a resolver for the given config.
func NewTexConvertResolver(cfg TexConvertConfig) *TexConvertResolver {
	return &TexConvertResolver{config: cfg}
}

// Resolve returns the first matching hint for the filename (case-insensitive).
// Matching is based on the hint Pattern wildcard and compares to the base filename.
func (r *TexConvertResolver) Resolve(name string) (TextureHint, bool) {
	return ResolveTexConvert(name, r.config)
}

// Resolve resolves a hint by filename using the provided config.
// This is a convenience wrapper around ResolveTexConvert.
func Resolve(name string, cfg TexConvertConfig) (TextureHint, bool) {
	return ResolveTexConvert(name, cfg)
}

// ResolveTexConvert resolves a hint by filename using the provided config.
func ResolveTexConvert(name string, cfg TexConvertConfig) (TextureHint, bool) {
	base := filepath.Base(name)
	lower := strings.ToLower(base)
	for _, hint := range cfg.Hints {
		pattern := strings.ToLower(hint.Pattern)
		if pattern == "" {
			continue
		}

		if wildcardMatch(pattern, lower) {
			return hint, true
		}
	}

	return TextureHint{}, false
}

// ResolveTexConvertDefault resolves using the current default config.
// It returns an error if the default config failed to load or initialize.
func ResolveTexConvertDefault(name string) (TextureHint, bool, error) {
	cfg, err := DefaultTexConvertConfig()
	if err != nil {
		return TextureHint{}, false, err
	}

	hint, ok := ResolveTexConvert(name, cfg)
	return hint, ok, nil
}

// ResolveOrFallback tries to resolve name against cfg;
// if unmatched, retries with "texture_"+fallbackSuffix+".paa" as a synthetic name.
// Returns the resolved name, its hint, and ok=false if both attempts fail.
func ResolveOrFallback(name, fallbackSuffix string, cfg TexConvertConfig) (string, TextureHint, bool) {
	if hint, ok := ResolveTexConvert(name, cfg); ok {
		return name, hint, true
	}

	fallbackName := "texture_" + fallbackSuffix + ".paa"
	if hint, ok := ResolveTexConvert(fallbackName, cfg); ok {
		return fallbackName, hint, true
	}

	return "", TextureHint{}, false
}

// wildcardMatch matches patterns with '*' and '?' against s (case-insensitive).
func wildcardMatch(pattern, s string) bool {
	p := []rune(pattern)
	str := []rune(s)
	pi, si := 0, 0
	starIdx := -1
	match := 0

	// Iterate over the string and pattern.
	for si < len(str) {
		// Match a single character.
		if pi < len(p) && (p[pi] == '?' || p[pi] == str[si]) {
			pi++
			si++
			continue
		}

		// Match a wildcard.
		if pi < len(p) && p[pi] == '*' {
			starIdx = pi
			match = si
			pi++
			continue
		}

		// Match a wildcard.
		if starIdx != -1 {
			pi = starIdx + 1
			match++
			si = match
			continue
		}

		// No match found.
		return false
	}

	// Check if the pattern is fully matched.
	for pi < len(p) && p[pi] == '*' {
		pi++
	}

	return pi == len(p)
}
