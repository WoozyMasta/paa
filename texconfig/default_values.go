package texconfig

// defaultTexConvertConfig returns the library default config mirroring TexConvert.cfg.
func defaultTexConvertConfig() TexConvertConfig {
	return TexConvertConfig{
		ConvertVersion:      6,
		UseSRGBFromDynRange: true,
		Hints: []TextureHint{
			{
				ClassName: "TexRGBA8888",
				Pattern:   "*_8888.*",
				// FIXME: TexView crashes on ARGB1555 for *_8888; disabled in encoder.
				Format:   TexFormatARGB1555,
				DynRange: boolPtr(true),
			},
			{
				ClassName: "ColorMap",
				Pattern:   "*_co.*",
				Format:    TexFormatDXT1,
				DynRange:  boolPtr(true),
			},
			{
				ClassName: "ColorMapRaw",
				Pattern:   "*_raw.*",
				// FIXME: TexView crashes on ARGB1555 for *_raw; disabled in encoder.
				Format:   TexFormatARGB1555,
				DynRange: boolPtr(false),
			},
			{
				ClassName: "ColorAlphaMap",
				Pattern:   "*_ca.*",
				Format:    TexFormatDXT5,
				DynRange:  boolPtr(true),
			},
			{
				ClassName:    "ColorAlphaTest",
				Pattern:      "*_cat.*",
				Format:       TexFormatDXT5,
				DynRange:     boolPtr(true),
				MipmapFilter: MipmapFilterAlphaNoise,
			},
			{
				ClassName: "sky",
				Pattern:   "*_sky.*",
				Format:    TexFormatDXT5,
				Swizzle:   swizzle("R", "1-A", "B", "1-G"),
				DynRange:  boolPtr(false),
			},
			{
				ClassName:    "detail",
				Pattern:      "*_detail.*",
				Format:       TexFormatDXT1,
				EnableDXT:    boolPtr(true),
				Swizzle:      swizzle("A", "A", "A", "1"),
				DynRange:     boolPtr(false),
				MipmapFilter: MipmapFilterFadeOut,
			},
			{
				ClassName:    "color_detail",
				Pattern:      "*_cdt.*",
				Format:       TexFormatDXT1,
				EnableDXT:    boolPtr(true),
				DynRange:     boolPtr(false),
				MipmapFilter: MipmapFilterFadeOut,
			},
			{
				ClassName:  "layer_color",
				Pattern:    "*_lco.*",
				Format:     TexFormatDXT1,
				EnableDXT:  boolPtr(true),
				AutoReduce: boolPtr(true),
				DynRange:   boolPtr(false),
			},
			{
				ClassName:  "multiply_color",
				Pattern:    "*_mco.*",
				Format:     TexFormatDXT1,
				EnableDXT:  boolPtr(true),
				AutoReduce: boolPtr(true),
				DynRange:   boolPtr(false),
			},
			{
				ClassName: "layer_color_draft",
				Pattern:   "*_draftlco.*",
				// FIXME: TexView crashes on ARGB1555 for *_draftlco; disabled in encoder.
				Format:    TexFormatARGB1555,
				EnableDXT: boolPtr(false),
				DynRange:  boolPtr(false),
			},
			{
				ClassName: "layer_color_alpha",
				Pattern:   "*_lca.*",
				Format:    TexFormatDXT5,
				EnableDXT: boolPtr(true),
				DynRange:  boolPtr(false),
			},
			{
				ClassName:  "mask",
				Pattern:    "*_mask.*",
				Format:     TexFormatDXT1,
				EnableDXT:  boolPtr(true),
				AutoReduce: boolPtr(true),
				DynRange:   boolPtr(false),
			},
			{
				ClassName:  "prt",
				Pattern:    "*_pr.*",
				Format:     TexFormatDXT5,
				EnableDXT:  boolPtr(true),
				AutoReduce: boolPtr(true),
				DynRange:   boolPtr(false),
			},
			{
				ClassName:    "ambient_diffuse_shadow",
				Pattern:      "*_ads.*",
				Format:       TexFormatDXT1,
				EnableDXT:    boolPtr(true),
				Swizzle:      swizzle("1", "G", "B", "1"),
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsDistance,
			},
			{
				ClassName:    "ambient_diffuse_shadow_hq",
				Pattern:      "*_adshq.*",
				Format:       TexFormatDXT5,
				Swizzle:      swizzle("0", "B", "0", "G"),
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsDistance,
			},
			{
				ClassName:    "detail_specular_diffuseinverse_map",
				Pattern:      "*_dtsmdi.*",
				Format:       TexFormatDXT1,
				EnableDXT:    boolPtr(true),
				Swizzle:      swizzle("R", "G", "B", "1"),
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsDistance,
			},
			{
				ClassName: "macro",
				Pattern:   "*_mc.*",
				Format:    TexFormatDXT5,
				EnableDXT: boolPtr(true),
				DynRange:  boolPtr(false),
			},
			{
				ClassName: "ambient_shadow",
				Pattern:   "*_as.*",
				Format:    TexFormatDXT1,
				EnableDXT: boolPtr(true),
				Swizzle:   swizzle("1", "G", "1", "1"),
				DynRange:  boolPtr(false),
			},
			{
				ClassName: "specular_map",
				Pattern:   "*_sm.*",
				Format:    TexFormatDXT1,
				EnableDXT: boolPtr(true),
				Swizzle:   swizzle("R", "G", "B", "1"),
				DynRange:  boolPtr(false),
			},
			{
				ClassName: "specular_diffuseinverse_map",
				Pattern:   "*_smdi.*",
				Format:    TexFormatDXT1,
				EnableDXT: boolPtr(true),
				Swizzle:   swizzle("1", "G", "B", "1"),
				DynRange:  boolPtr(false),
			},
			{
				ClassName:    "detail_short",
				Pattern:      "*_dt.*",
				Format:       TexFormatDXT1,
				EnableDXT:    boolPtr(true),
				Swizzle:      swizzle("A", "A", "A", "1"),
				DynRange:     boolPtr(false),
				MipmapFilter: MipmapFilterFadeOut,
			},
			{
				ClassName:    "normalmap",
				Pattern:      "*_normalmap.*",
				EnableDXT:    boolPtr(true),
				MipmapFilter: MipmapFilterNormalizeNormalMapAlpha,
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
			},
			{
				ClassName:    "normalmap_short",
				Pattern:      "*_no.*",
				Format:       TexFormatDXT1,
				EnableDXT:    boolPtr(true),
				MipmapFilter: MipmapFilterNormalizeNormalMapAlpha,
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
				Swizzle:      swizzle("", "", "", "1"),
			},
			{
				ClassName:    "normalmap_uncompressed",
				Pattern:      "*_noex.*",
				Format:       TexFormatARGB4444,
				EnableDXT:    boolPtr(false),
				MipmapFilter: MipmapFilterNormalizeNormalMapAlpha,
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
			},
			{
				ClassName:    "NormalMapNoise",
				Pattern:      "*_non.*",
				Format:       TexFormatDXT5,
				MipmapFilter: MipmapFilterNormalizeNormalMapNoise,
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
			},
			{
				ClassName:    "normalmap_hq",
				Pattern:      "*_nohq.*",
				Format:       TexFormatDXT5,
				Swizzle:      swizzle("1-A", "G", "B", "1-R"),
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
				MipmapFilter: MipmapFilterNormalizeNormalMapAlpha,
			},
			{
				ClassName:    "normalmap_vhq",
				Pattern:      "*_novhq.*",
				Format:       TexFormatDXT5,
				Swizzle:      swizzle("1", "G", "1", "1-R"),
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
				MipmapFilter: MipmapFilterNormalizeNormalMapAlpha,
			},
			{
				ClassName:    "normalmap_hq_fade",
				Pattern:      "*_nofhq.*",
				Format:       TexFormatDXT5,
				Swizzle:      swizzle("1-A", "G", "B", "1-R"),
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
				MipmapFilter: MipmapFilterNormalizeNormalMapFade,
			},
			{
				ClassName:    "normalmapFade",
				Pattern:      "*_nof.*",
				EnableDXT:    boolPtr(true),
				MipmapFilter: MipmapFilterNormalizeNormalMapFade,
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
			},
			{
				ClassName:    "normalmapFade_uncompressed",
				Pattern:      "*_nofex.*",
				Format:       TexFormatARGB4444,
				EnableDXT:    boolPtr(false),
				MipmapFilter: MipmapFilterNormalizeNormalMapFade,
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
			},
			{
				ClassName:    "normalmap_spec",
				Pattern:      "*_ns.*",
				EnableDXT:    boolPtr(true),
				MipmapFilter: MipmapFilterNormalizeNormalMap,
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
			},
			{
				ClassName:    "normalmap_parallax",
				Pattern:      "*_nopx.*",
				Format:       TexFormatDXT5,
				EnableDXT:    boolPtr(true),
				MipmapFilter: MipmapFilterNormalizeNormalMap,
				DynRange:     boolPtr(false),
				Swizzle:      swizzle("A", "G", "", "1-R"),
				ErrorMetrics: ErrorMetricsNormalMap,
			},
			{
				ClassName:    "normalmap_spec_uncompressed",
				Pattern:      "*_nsex.*",
				Format:       TexFormatARGB4444,
				EnableDXT:    boolPtr(false),
				MipmapFilter: MipmapFilterNormalizeNormalMap,
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
			},
			{
				ClassName:    "normalmap_spec_hq",
				Pattern:      "*_nshq.*",
				Format:       TexFormatDXT5,
				Swizzle:      swizzle("1-A", "G", "B", "1-R"),
				DynRange:     boolPtr(false),
				ErrorMetrics: ErrorMetricsNormalMap,
				MipmapFilter: MipmapFilterNormalizeNormalMapAlpha,
			},
			{
				ClassName: "grayscalealpha",
				Pattern:   "*_gs.*",
				Format:    TexFormatAI88,
			},
			{
				ClassName:    "AddAlphaNoise",
				Pattern:      "*_can.*",
				Format:       TexFormatDXT5,
				MipmapFilter: MipmapFilterAddAlphaNoise,
				DynRange:     boolPtr(true),
				ErrorMetrics: ErrorMetricsDistance,
			},
			{
				ClassName: "TexRGBA4444",
				Pattern:   "*_4444.*",
				Format:    TexFormatARGB4444,
				DynRange:  boolPtr(false),
			},
			{
				ClassName: "TexRGBA1555",
				Pattern:   "*_1555.*",
				Format:    TexFormatARGB1555,
				DynRange:  boolPtr(false),
			},
			{
				ClassName: "TexAI88",
				Pattern:   "*_88.*",
				Format:    TexFormatAI88,
				DynRange:  boolPtr(false),
			},
			{
				ClassName: "TexDXT1",
				Pattern:   "*_dxt1.*",
				Format:    TexFormatDXT1,
				DynRange:  boolPtr(false),
			},
			{
				ClassName: "TexDXT5",
				Pattern:   "*_dxt5.*",
				Format:    TexFormatDXT5,
				DynRange:  boolPtr(false),
			},
		},
	}
}

// boolPtr converts a boolean to a pointer to a boolean.
func boolPtr(v bool) *bool {
	return &v
}

// swizzle converts a string to a ChannelSwizzle.
func swizzle(r, g, b, a string) ChannelSwizzle {
	var s ChannelSwizzle
	if r != "" {
		s.R = mustSwizzleExpr(r)
	}
	if g != "" {
		s.G = mustSwizzleExpr(g)
	}
	if b != "" {
		s.B = mustSwizzleExpr(b)
	}
	if a != "" {
		s.A = mustSwizzleExpr(a)
	}

	return s
}

// mustSwizzleExpr converts a string to a SwizzleExpr.
func mustSwizzleExpr(s string) SwizzleExpr {
	expr, err := ParseSwizzleExpr(s)
	if err != nil {
		panic(err)
	}

	return expr
}
