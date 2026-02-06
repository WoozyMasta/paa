package paa

import (
	"image"
	"image/color"
	"io"

	"github.com/woozymasta/bcn"
	"github.com/woozymasta/paa/texconfig"
)

// EncodeWithTexConfig resolves filename-based settings from a TexConvert config
// and encodes the image using those settings. If no hint matches, it falls back
// to EncodeWithOptions with auto format selection.
func EncodeWithTexConfig(w io.Writer, img image.Image, name string, cfg texconfig.TexConvertConfig) error {
	return EncodeWithTexConfigOptions(w, img, name, cfg, nil)
}

// EncodeWithTexConfigOptions resolves filename-based settings from a TexConvert config,
// applies optional overrides, and encodes the image using those settings.
func EncodeWithTexConfigOptions(w io.Writer, img image.Image, name string, cfg texconfig.TexConvertConfig, override *EncodeOptions) error {
	hint, ok := texconfig.Resolve(name, cfg)
	if !ok {
		opts := &EncodeOptions{}
		if !cfg.DisableLZO {
			opts.UseLZO = true
		}
		opts.ForceCXAMFull = isDXT(opts.Type) || opts.Type == 0
		if cfg.ApplyDefaultErrorMetrics {
			ensureBCnOptions(opts).RGBWeights = &bcn.RGBWeights{R: 5, G: 9, B: 2}
		}
		if override != nil {
			applyEncodeOverrides(opts, override)
		}

		return EncodeWithOptions(w, img, opts)
	}

	skipSwizzle := shouldSkipSwizzle(img, hint)
	if !skipSwizzle {
		img = promoteAlphaFromRGBIfNeeded(img, hint)
	}

	img = autoReduceIfNeeded(img, hint, cfg)
	opts, err := EncodeOptionsFromHint(img, hint, cfg, skipSwizzle)
	if err != nil {
		return err
	}

	if override != nil {
		applyEncodeOverrides(opts, override)
	}

	return EncodeWithOptions(w, img, opts)
}

// EncodeOptionsFromHint converts a resolved TexConvert hint into EncodeOptions.
func EncodeOptionsFromHint(img image.Image, hint texconfig.TextureHint, cfg texconfig.TexConvertConfig, skipSwizzle bool) (*EncodeOptions, error) {
	stats := scanAlpha(img)

	if isTexViewUnsupported(hint) {
		return nil, ErrUnsupportedFormat
	}

	if hint.EnableDXT != nil && !*hint.EnableDXT {
		switch hint.Format {
		case texconfig.TexFormatARGB4444, texconfig.TexFormatARGB1555, texconfig.TexFormatAI88, texconfig.TexFormatP8:
			// Allow explicit non-DXT formats when DXT is disabled.
		default:
			return nil, ErrUnsupportedFormat
		}
	}

	paxType, err := selectPaxType(stats, hint)
	if err != nil {
		return nil, err
	}

	opts := &EncodeOptions{Type: paxType}
	if isDXT(paxType) && !cfg.DisableLZO {
		opts.UseLZO = true
	}
	if isDXT(paxType) {
		opts.ForceCXAMFull = true
	}
	if paxType == PaxARGB4 {
		opts.ForceCXAMFull = true
		opts.ForceLZSS = true
	}
	if paxType == PaxARGBA5 || paxType == PaxARGB8 {
		opts.ForceCXAMFull = true
	}

	// Apply swizzle.
	if !hint.Swizzle.IsIdentity() {
		vs := hint.VirtualSwz == nil || *hint.VirtualSwz
		if vs {
			tag, ok, err := hint.Swizzle.ZIWSTag()
			if err != nil {
				return nil, err
			}
			if ok {
				opts.WriteSwizzleTag = true
				opts.SwizzleTag = tag
			}
		}
		if !skipSwizzle {
			opts.Swizzle = &hint.Swizzle
		} else {
			opts.SkipSwizzle = true
		}
	}

	if !stats.allHigh {
		opts.WriteGALF = true
		if stats.isBinary {
			opts.GALFValue = 2
		} else {
			opts.GALFValue = 1
		}
	}

	// Apply GALF.
	if isDetailHint(hint) {
		opts.WriteGALF = true
		opts.GALFValue = 2
	}

	// Apply error metrics.
	switch hint.ErrorMetrics {
	case texconfig.ErrorMetricsDistance:
		ensureBCnOptions(opts).RGBWeights = &bcn.RGBWeights{R: 5, G: 5, B: 0}
	case texconfig.ErrorMetricsNormalMap:
		ensureBCnOptions(opts).RGBWeights = &bcn.RGBWeights{R: 5, G: 5, B: 5}
	case texconfig.ErrorMetricsDefault:
		if cfg.ApplyDefaultErrorMetrics {
			ensureBCnOptions(opts).RGBWeights = &bcn.RGBWeights{R: 5, G: 9, B: 2}
		}

	// save all by default because dont know what is it (maybe normal map or some special format with strong B channel)
	default:
		ensureBCnOptions(opts).RGBWeights = &bcn.RGBWeights{R: 5, G: 5, B: 5}
	}

	// Apply mipmap filter.
	if hint.MipmapFilter != texconfig.MipmapFilterDefault {
		filter := hint.MipmapFilter
		opts.MipmapFilter = &filter
	}
	if cfg.UseSRGBFromDynRange && hint.DynRange != nil && *hint.DynRange {
		opts.UseSRGB = true
	}

	return opts, nil
}

// alphaStats is a struct that contains information about the alpha channel of an image.
type alphaStats struct {
	hasAlpha bool
	allHigh  bool
	isBinary bool
}

// scanAlpha inspects the image alpha channel for GALF/format decisions.
func scanAlpha(img image.Image) alphaStats {
	b := img.Bounds()
	stats := alphaStats{allHigh: true, isBinary: true}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			a8 := uint8(a >> 8) //nolint:gosec // G115
			if a8 < 255 {
				stats.hasAlpha = true
			}
			if a8 < 0xF0 {
				stats.allHigh = false
			}
			if a8 != 0 && a8 != 255 {
				stats.isBinary = false
			}
		}
	}

	// If the image has no alpha, set the alpha channel to binary.
	if !stats.hasAlpha {
		stats.isBinary = true
	}

	return stats
}

// selectPaxType selects the PaxType based on the alpha channel and the hint.
func selectPaxType(stats alphaStats, hint texconfig.TextureHint) (PaxType, error) {
	switch hint.Format {
	// Default format.
	case texconfig.TexFormatDefault:
		// If the image has no alpha, set the alpha channel to binary.
		if stats.allHigh || !stats.hasAlpha || stats.isBinary {
			return PaxDXT1, nil
		}
		return PaxDXT5, nil

	// Supported DXT formats.
	case texconfig.TexFormatDXT1:
		return PaxDXT1, nil
	case texconfig.TexFormatDXT5:
		return PaxDXT5, nil

	// Unsupported DXT formats.
	case texconfig.TexFormatDXT2, texconfig.TexFormatDXT3, texconfig.TexFormatDXT4:
		return 0, ErrUnsupportedFormat

	// Supported non-DXT formats.
	case texconfig.TexFormatARGB4444:
		return PaxARGB4, nil
	case texconfig.TexFormatARGB1555:
		return PaxARGBA5, nil
	case texconfig.TexFormatAI88:
		return PaxGRAYA, nil

	// Unsupported non-DXT formats.
	case texconfig.TexFormatP8:
		return 0, ErrUnsupportedFormat
	default:
		return 0, ErrUnsupportedFormat
	}
}

// autoReduceIfNeeded reduces the image if the hint requires it.
func autoReduceIfNeeded(img image.Image, hint texconfig.TextureHint, cfg texconfig.TexConvertConfig) image.Image {
	if cfg.DisableAutoReduce || hint.AutoReduce == nil || !*hint.AutoReduce {
		return img
	}

	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return img
	}

	if hint.LimitSize <= 0 {
		return img
	}

	maxDim := b.Dx()
	if b.Dy() > maxDim {
		maxDim = b.Dy()
	}
	if maxDim <= hint.LimitSize {
		return img
	}

	useSRGB := cfg.UseSRGBFromDynRange && hint.DynRange != nil && *hint.DynRange
	mips := bcn.GenerateMipmaps(img, useSRGB)
	// Pick the largest mip that fits within limitSize.
	var best image.Image
	for _, m := range mips {
		bm := m.Bounds()
		if bm.Dx() <= hint.LimitSize && bm.Dy() <= hint.LimitSize {
			best = m
			break
		}
	}

	if best != nil {
		return best
	}

	return img
}

// applyEncodeOverrides applies the overrides to the EncodeOptions.
func applyEncodeOverrides(dst *EncodeOptions, override *EncodeOptions) {
	if override == nil || dst == nil {
		return
	}

	if override.BCn != nil {
		dst.BCn = mergeBCnOptions(dst.BCn, override.BCn)
	}
	if override.SkipSwizzle {
		dst.SkipSwizzle = true
		dst.Swizzle = nil
	}

	// Explicitly propagate ForceCXAMFull override.
	dst.ForceCXAMFull = override.ForceCXAMFull
	if override.ForceLZSS {
		dst.ForceLZSS = true
	}
}

// ensureBCnOptions ensures that the BCn options are set.
func ensureBCnOptions(opts *EncodeOptions) *bcn.EncodeOptions {
	if opts == nil {
		return nil
	}

	if opts.BCn == nil {
		opts.BCn = &bcn.EncodeOptions{}
	}

	return opts.BCn
}

// mergeBCnOptions merges the BCn options.
func mergeBCnOptions(dst *bcn.EncodeOptions, override *bcn.EncodeOptions) *bcn.EncodeOptions {
	if override == nil {
		return dst
	}

	if dst == nil {
		clone := *override
		if override.Refinement != nil {
			ref := *override.Refinement
			clone.Refinement = &ref
		}
		return &clone
	}

	if override.RGBWeights != nil {
		dst.RGBWeights = override.RGBWeights
	}
	if override.Refinement != nil {
		ref := *override.Refinement
		dst.Refinement = &ref
	}
	if override.QualityLevel != 0 {
		dst.QualityLevel = override.QualityLevel
	}
	if override.Workers != 0 {
		dst.Workers = override.Workers
	}
	if override.GenerateMipmaps {
		dst.GenerateMipmaps = true
	}
	if override.UseSRGB {
		dst.UseSRGB = true
	}
	if override.AlphaThreshold != 0 {
		dst.AlphaThreshold = override.AlphaThreshold
	}

	return dst
}

// shouldSkipSwizzle checks if the swizzle should be skipped.
func shouldSkipSwizzle(img image.Image, hint texconfig.TextureHint) bool {
	if hint.Swizzle.IsIdentity() {
		return false
	}

	if usesAlphaForRGB(hint.Swizzle) {
		minA, maxA, minRGB, maxRGB := alphaAndRGBRange(img)
		return minA == 255 && maxA == 255 && minRGB != maxRGB
	}
	return false
}

// isDetailHint checks if the hint is a detail hint.
func isDetailHint(hint texconfig.TextureHint) bool {
	switch hint.ClassName {
	case "detail", "detail_short":
		return true
	default:
		return false
	}
}

// isTexViewUnsupported marks formats that crash TexView despite matching TexConvert.cfg.
func isTexViewUnsupported(hint texconfig.TextureHint) bool {
	switch hint.ClassName {
	case "TexRGBA8888", "ColorMapRaw", "layer_color_draft":
		// FIXME: TexView crashes on ARGB1555 for *_raw and *_draftlco, and on *_8888 (mapped to ARGB1555 in TexConvert.cfg).
		return true
	default:
		return false
	}
}

// promoteAlphaFromRGBIfNeeded promotes the alpha channel from the RGB channel if needed.
func promoteAlphaFromRGBIfNeeded(img image.Image, hint texconfig.TextureHint) image.Image {
	if !usesAlphaForRGB(hint.Swizzle) {
		return img
	}

	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return img
	}

	minA, maxA, minRGB, maxRGB := alphaAndRGBRange(img)
	if minA != 255 || maxA != 255 {
		return img
	}

	if minRGB == maxRGB {
		return img
	}

	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			l := uint8((uint16(c.R) + uint16(c.G) + uint16(c.B)) / 3) //nolint:gosec // G115: clamped by uint16 sum
			out.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: l})
		}
	}

	return out
}

// usesAlphaForRGB checks if the swizzle uses the alpha channel for the RGB channels.
func usesAlphaForRGB(swz texconfig.ChannelSwizzle) bool {
	return swz.R.Valid && swz.G.Valid && swz.B.Valid &&
		swz.R.Source == texconfig.SwizzleA && !swz.R.Invert && !swz.R.IsConst &&
		swz.G.Source == texconfig.SwizzleA && !swz.G.Invert && !swz.G.IsConst &&
		swz.B.Source == texconfig.SwizzleA && !swz.B.Invert && !swz.B.IsConst &&
		swz.A.Valid && swz.A.IsConst && swz.A.ConstValue == 255
}

// alphaAndRGBRange checks the alpha and RGB channel ranges.
func alphaAndRGBRange(img image.Image) (minA, maxA uint8, minRGB, maxRGB uint8) {
	minA, maxA = 255, 0
	minRGB, maxRGB = 255, 0
	b := img.Bounds()

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A < minA {
				minA = c.A
			}
			if c.A > maxA {
				maxA = c.A
			}
			if c.R < minRGB {
				minRGB = c.R
			}
			if c.G < minRGB {
				minRGB = c.G
			}
			if c.B < minRGB {
				minRGB = c.B
			}
			if c.R > maxRGB {
				maxRGB = c.R
			}
			if c.G > maxRGB {
				maxRGB = c.G
			}
			if c.B > maxRGB {
				maxRGB = c.B
			}
		}
	}

	return minA, maxA, minRGB, maxRGB
}
