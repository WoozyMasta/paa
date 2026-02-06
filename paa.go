/*
Package paa reads and writes PAA (Arma/DayZ) texture files.

High-level usage:
  - DecodePAA reads a full PAA structure (tags + mipmaps).
  - Decode/DecodeConfig implement image.Decode/image.DecodeConfig.
  - Encode/EncodeWithOptions write a file with tags + mipmaps.
  - To register with image.Decode, import the img subpackage:
    _ "github.com/woozymasta/paa/img".

File structure (simplified):

	[2 bytes pax type] [GGAT tags...] [mipmap blocks...] [0,0,0,0,0,0]

Tags are optional in theory but always present in practice. Tag names are stored
as four-byte identifiers (e.g. "CGVA", "CXAM", "GALF", "ZIWS", "SFFO") and map to
avg/max/flags/swizzle/offsets. This package writes tags in the canonical order
used by BI tools: CGVA, CXAM, GALF (optional), ZIWS (optional), SFFO.

Mipmaps:
Each mipmap block is:

	uint16 width, uint16 height, 3-byte size, [size bytes payload]

Some files include a trailing “dummy” mipmap (width=0,height=0), but decoding
uses SFFO as the authoritative list of mip offsets.

DXT payload size is the *encoded* size (BC1/BC3 blocks), not width*height*4.
For DXT1: ((w+3)/4)*((h+3)/4)*8
For DXT5: ((w+3)/4)*((h+3)/4)*16

LZO compression:
Arma2+ allows per-mip LZO compression for DXT formats. The top bit of the
mip width indicates LZO-compressed data. This package uses per-mip LZO only
when it reduces size; otherwise the mip is stored raw. Always mask width
(width & 0x7FFF) for dimension calculations.

SFFO (offset table):
SFFO contains 16 uint32 offsets to mipmap blocks, relative to file start.
Only as many entries as actual mip levels are filled; remaining entries are 0.
The engine can derive offsets without SFFO, but BI tools always write it.

Normal maps (_nohq):
Arma stores tangent-space normal maps in DXT5 with a swizzle tag:

	SWIZTAGG (ZIWS) = 0x05 0x04 0x02 0x03

Channels are interpreted as:

	stored R=0, G=Y, B=Z, A=255-X

The runtime reconstructs display channels as:

	X = 255 - A, Y = G, Z = B

This package can apply the swizzle on encode (e.g. NormalMapSwizzle or hint‑driven
swizzle) and will unswizzle on decode when the ZIWS tag matches. For some hints
we emit ZIWS but intentionally keep payload unswizzled to avoid double‑swizzle
in external tools.

Important caveat: some external viewers ignore SWIZTAGG and display raw
channels, which makes _nohq appear incorrect even when the data is valid.

CGVA/CXAM tag order and encoding:
BI tools write these tags in BGRA order (not RGBA). For _nohq, CXAM is
always FF FF FF FF. These details affect TexView’s PNG export.
EncodeWithOptions writes CGVA/CXAM in BGRA and matches BI behavior.

Mipmap defaults:
EncodeWithOptions generates a full mip chain down to 4x4 (BI default) unless
overridden. GUI textures often disable mips; use EncodeOptions.GenerateMipmaps=false
or MaxMipCount=1.

Format pitfalls and decisions:
  - Tag order matters for some tools.
  - SWIZTAGG is required for correct _nohq interpretation.
  - LZO is per-mip and signaled by width’s top bit.
  - The file may be valid even if some viewers show wrong colors (swizzle ignored).

See EncodeOptions for knobs controlling quality, swizzle, LZO, and mipmaps.

TexConvert.cfg support:
The texconfig package provides a default config mirroring BI’s TexConvert.cfg and
filename-based hint resolution. Some legacy formats are intentionally rejected
because TexView crashes on them (e.g. *_raw, *_draftlco, *_8888).
*/
package paa

// PAA represents a .paa texture file (Arma/Bohemia format).
// For use with image.Decode, import the img subpackage: _ "github.com/woozymasta/paa/img".
type PAA struct {
	// Tag identifiers (e.g. "CGVA", "CXAM", "GALF", "ZIWS", "SFFO") are stored as-is.
	// This package writes tags in the canonical order used by BI tools: CGVA, CXAM, GALF (optional), ZIWS (optional), SFFO.
	Taggs map[string][]byte

	// Mipmaps are stored in the file in order.
	MipMaps []*MipMap

	// Palette is stored in the file.
	Palette []byte

	// Type is the PaxType of the texture.
	Type PaxType
}
