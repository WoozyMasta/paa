package paa

import (
	"github.com/woozymasta/bcn"
	"github.com/woozymasta/paa/texconfig"
)

// EncodeOptions configures PAA encoding (format, normal-map handling, and BCn options).
// Used by EncodeWithOptions and EncodeOptionsFromHint.
type EncodeOptions struct {
	// BCn overrides DXT/BCn encoding options (quality, refinement, workers).
	// Nil uses bcn defaults.
	BCn *bcn.EncodeOptions
	// Swizzle applies a generic channel swizzle before encoding (TexConvert.cfg style).
	Swizzle *texconfig.ChannelSwizzle
	// GenerateMipmaps controls mipmap generation. Nil = default (true).
	GenerateMipmaps *bool
	// MipmapFilter selects a specific mipmap filter (TexConvert.cfg).
	MipmapFilter *texconfig.MipmapFilter
	// MaxMipCount limits the number of mip levels (including base). 0 = no limit.
	MaxMipCount int
	// MinMipSize stops mip generation when both dimensions are <= this value. 0 = default (4).
	MinMipSize int
	// Type is the PAA pixel format (PaxDXT1, PaxDXT5, etc.).
	// Zero value means auto: DXT5 if image has any non-opaque alpha, else DXT1.
	Type PaxType
	// SwizzleTag is the SWIZTAGG payload to write when WriteSwizzleTag is true.
	SwizzleTag [4]byte
	// NormalMapSwizzle, if true, applies swizzleNormalMap to the image before encoding
	// and forces DXT5 (for _nohq normal maps). Ignored if Type is set to non-DXT.
	NormalMapSwizzle bool
	// WriteNohqSwizzleTag when true writes SWIZTAGG (0x05040203) so Arma/original tools
	// interpret DXT5 channels as nohq normal map (R=255-A, G=G, B=B, A=255-R).
	WriteNohqSwizzleTag bool
	// WriteSwizzleTag writes a SWIZTAGG payload (not limited to nohq).
	WriteSwizzleTag bool
	// SkipSwizzle disables applying Swizzle to the pixel payload (useful when input
	// is already swizzled, e.g. PAA -> PAA roundtrips).
	SkipSwizzle bool
	// WriteGALF writes a GALF tag. If false and opts is provided, no GALF is written.
	WriteGALF bool
	// GALFValue is the GALF payload byte (typically 1; detail maps use 2).
	GALFValue byte
	// ForceCXAMFull, when true, writes CXAM as 0xFF in all channels regardless of actual max.
	// This matches BI tools for DXT payloads and aligns TexView "max color" stats.
	ForceCXAMFull bool
	// UseLZO enables Arma2+ LZO compression for DXT payloads (width high bit).
	UseLZO bool
	// ForceLZSS forces LZSS compression for non-DXT payloads, even when it grows size.
	ForceLZSS bool
	// UseSRGB enables sRGB-aware downscale for mip generation.
	UseSRGB bool
}

// DecodeOptions configures PAA decoding.
// BCn options are forwarded to the BCn decoder (e.g. workers).
type DecodeOptions struct {
	// BCn overrides DXT/BCn decoding options (workers).
	// Nil uses bcn defaults.
	BCn *bcn.DecodeOptions
}

// Note: filename-based resolution is provided by the texconfig package.
