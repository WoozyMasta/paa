/*
Package texconfig implements TexConvert.cfg-compatible configuration for PAA encoding.

It provides a typed config model, a parser for the original cfg syntax, and a
resolver that matches filename patterns to effective encoding hints (format,
swizzle, mipmap filter, error metrics, and related options). The structures
are tagged for JSON/YAML so tools can persist and override configs without
using the cfg format.

The default config mirrors the original TexConvert.cfg values, but the package
also exposes a few optional extensions (e.g. global flags that tweak AutoReduce
or sRGB handling). These are intentionally explicit and opt-in.
*/
package texconfig

// TexConvertConfig is a typed representation of TexConvert.cfg.
// It is safe to modify a copy and pass into Resolve for per-tool overrides.
type TexConvertConfig struct {
	// Hints is the flattened list of TextureHints in the order they appear in config.
	Hints []TextureHint `json:"hints" yaml:"hints"`

	// ConvertVersion is the config file version (TexConvert.cfg: convertVersion).
	ConvertVersion int `json:"convert_version" yaml:"convert_version"`

	// ApplyDefaultErrorMetrics forces default error weights when hint.ErrorMetrics is Default.
	// This is an extension over TexConvert.cfg behavior.
	ApplyDefaultErrorMetrics bool `json:"apply_default_error_metrics,omitempty" yaml:"apply_default_error_metrics,omitempty"`

	// UseSRGBFromDynRange applies DynRange to mipmap generation (sRGB downscale).
	// This is an extension over TexConvert.cfg behavior.
	UseSRGBFromDynRange bool `json:"use_srgb_from_dyn_range,omitempty" yaml:"use_srgb_from_dyn_range,omitempty"`

	// DisableAutoReduce disables AutoReduce even when hints request it.
	// This is an extension over TexConvert.cfg behavior.
	DisableAutoReduce bool `json:"disable_autoreduce,omitempty" yaml:"disable_autoreduce,omitempty"`

	// DisableLZO disables LZO compression for DXT payloads.
	// This is an extension over TexConvert.cfg behavior.
	DisableLZO bool `json:"disable_lzo,omitempty" yaml:"disable_lzo,omitempty"`
}

// TextureHint represents a single class entry in TexConvert.cfg (resolved and flattened).
// Fields map directly to TexConvert.cfg properties.
type TextureHint struct {
	// EnableDXT toggles whether DXT is allowed for this hint.
	EnableDXT *bool `json:"enable_dxt,omitempty" yaml:"enable_dxt,omitempty"`

	// DynRange toggles dynamic range (used by original tools).
	DynRange *bool `json:"dyn_range,omitempty" yaml:"dyn_range,omitempty"`

	// AutoReduce enables auto downscale before compression.
	AutoReduce *bool `json:"autoreduce,omitempty" yaml:"autoreduce,omitempty"`

	// VirtualSwz enables writing ZIWS/SWIZTAGG when swizzle is non-identity.
	VirtualSwz *bool `json:"virtual_swizzle,omitempty" yaml:"virtual_swizzle,omitempty"`

	// Dithering toggles dithering (used by original tools).
	Dithering *bool `json:"dithering,omitempty" yaml:"dithering,omitempty"`

	// ClassName is the config class name (used for diagnostics).
	ClassName string `json:"class_name" yaml:"class_name"`

	// Extends is the base class name (if any).
	Extends string `json:"extends,omitempty" yaml:"extends,omitempty"`

	// Pattern is a filename wildcard (e.g. "*_nohq*") used by Resolve.
	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`

	// Swizzle describes the channel remap to apply before encoding.
	Swizzle ChannelSwizzle `json:"swizzle,omitempty" yaml:"swizzle,omitempty"`

	// Format is the requested encoder format (TexFormDefault enum).
	Format TexFormat `json:"format,omitempty" yaml:"format,omitempty"`

	// MipmapFilter is the mip filter name (TexMipDefault enum).
	MipmapFilter MipmapFilter `json:"mipmap_filter,omitempty" yaml:"mipmap_filter,omitempty"`

	// ErrorMetrics selects RGB error weights for DXT compression.
	ErrorMetrics ErrorMetrics `json:"error_metrics,omitempty" yaml:"error_metrics,omitempty"`

	// LimitSize is the max dimension limit from config (0 = no limit).
	LimitSize int `json:"limit_size,omitempty" yaml:"limit_size,omitempty"`
}

// Clone returns a deep copy of the config.
func (c TexConvertConfig) Clone() TexConvertConfig {
	out := TexConvertConfig{
		ConvertVersion: c.ConvertVersion,
		Hints:          make([]TextureHint, len(c.Hints)),
	}
	copy(out.Hints, c.Hints)

	return out
}
