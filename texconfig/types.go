package texconfig

import (
	"errors"
	"fmt"
	"strings"
)

// TexFormat describes the target PAA pixel format requested by TexConvert.cfg.
type TexFormat int

// TexFormat values mirror TexConvert.cfg format names.
const (
	TexFormatDefault  TexFormat = iota // Default format.
	TexFormatP8                        // P8 format.
	TexFormatARGB4444                  // ARGB4444 format.
	TexFormatARGB1555                  // ARGB1555 format.
	TexFormatAI88                      // AI88 format.
	TexFormatDXT1                      // DXT1 format.
	TexFormatDXT2                      // DXT2 format.
	TexFormatDXT3                      // DXT3 format.
	TexFormatDXT4                      // DXT4 format.
	TexFormatDXT5                      // DXT5 format.
)

// String returns the string representation of the TexFormat.
func (f TexFormat) String() string {
	switch f {
	case TexFormatDefault:
		return "Default"
	case TexFormatP8:
		return "P8"
	case TexFormatARGB4444:
		return "ARGB4444"
	case TexFormatARGB1555:
		return "ARGB1555"
	case TexFormatAI88:
		return "AI88"
	case TexFormatDXT1:
		return "DXT1"
	case TexFormatDXT2:
		return "DXT2"
	case TexFormatDXT3:
		return "DXT3"
	case TexFormatDXT4:
		return "DXT4"
	case TexFormatDXT5:
		return "DXT5"
	default:
		return fmt.Sprintf("TexFormat(%d)", int(f))
	}
}

// ParseTexFormat converts a config string to a TexFormat.
func ParseTexFormat(s string) (TexFormat, bool) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.ToUpper(s)

	if s == "" || s == "DEFAULT" {
		return TexFormatDefault, true
	}

	switch s {
	case "P8":
		return TexFormatP8, true
	case "ARGB4444":
		return TexFormatARGB4444, true
	case "ARGB1555":
		return TexFormatARGB1555, true
	case "AI88":
		return TexFormatAI88, true
	case "DXT1":
		return TexFormatDXT1, true
	case "DXT2":
		return TexFormatDXT2, true
	case "DXT3":
		return TexFormatDXT3, true
	case "DXT4":
		return TexFormatDXT4, true
	case "DXT5":
		return TexFormatDXT5, true
	default:
		return TexFormatDefault, false
	}
}

// MarshalText implements encoding.TextMarshaler.
func (f TexFormat) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (f *TexFormat) UnmarshalText(text []byte) error {
	val, ok := ParseTexFormat(string(text))
	if !ok {
		return fmt.Errorf("unknown TexFormat %q", string(text))
	}

	*f = val

	return nil
}

// MipmapFilter describes mip generation behavior.
type MipmapFilter int

// MipmapFilter values mirror TexConvert.cfg mipmapFilter names.
const (
	MipmapFilterDefault                 MipmapFilter = iota // Default mipmap filter.
	MipmapFilterFadeOutAlpha                                // Fade out alpha mipmap filter.
	MipmapFilterNormalizeNormalMap                          // Normalize normal map mipmap filter.
	MipmapFilterFadeOut                                     // Fade out mipmap filter.
	MipmapFilterAlphaNoise                                  // Alpha noise mipmap filter.
	MipmapFilterNormalizeNormalMapAlpha                     // Normalize normal map alpha mipmap filter.
	MipmapFilterNormalizeNormalMapNoise                     // Normalize normal map noise mipmap filter.
	MipmapFilterNormalizeNormalMapFade                      // Normalize normal map fade mipmap filter.
	MipmapFilterAddAlphaNoise                               // Add alpha noise mipmap filter.
)

// String returns the string representation of the MipmapFilter.
func (f MipmapFilter) String() string {
	switch f {
	case MipmapFilterDefault:
		return "Default"
	case MipmapFilterFadeOutAlpha:
		return "FadeOutAlpha"
	case MipmapFilterNormalizeNormalMap:
		return "NormalizeNormalMap"
	case MipmapFilterFadeOut:
		return "FadeOut"
	case MipmapFilterAlphaNoise:
		return "AlphaNoise"
	case MipmapFilterNormalizeNormalMapAlpha:
		return "NormalizeNormalMapAlpha"
	case MipmapFilterNormalizeNormalMapNoise:
		return "NormalizeNormalMapNoise"
	case MipmapFilterNormalizeNormalMapFade:
		return "NormalizeNormalMapFade"
	case MipmapFilterAddAlphaNoise:
		return "AddAlphaNoise"
	default:
		return fmt.Sprintf("MipmapFilter(%d)", int(f))
	}
}

// ParseMipmapFilter converts a config string to a MipmapFilter.
// It trims the string, removes quotes, and converts to lowercase.
func ParseMipmapFilter(s string) (MipmapFilter, bool) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.ToLower(s)

	if s == "" || s == "default" {
		return MipmapFilterDefault, true
	}

	switch s {
	case "fadeoutalpha":
		return MipmapFilterFadeOutAlpha, true
	case "normalizenormalmap":
		return MipmapFilterNormalizeNormalMap, true
	case "fadeout":
		return MipmapFilterFadeOut, true
	case "alphanoise":
		return MipmapFilterAlphaNoise, true
	case "normalizenormalmapalpha":
		return MipmapFilterNormalizeNormalMapAlpha, true
	case "normalizenormalmapnoise":
		return MipmapFilterNormalizeNormalMapNoise, true
	case "normalizenormalmapfade":
		return MipmapFilterNormalizeNormalMapFade, true
	case "addalphanoise":
		return MipmapFilterAddAlphaNoise, true
	default:
		return MipmapFilterDefault, false
	}
}

// MarshalText implements encoding.TextMarshaler.
func (f MipmapFilter) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (f *MipmapFilter) UnmarshalText(text []byte) error {
	val, ok := ParseMipmapFilter(string(text))
	if !ok {
		return fmt.Errorf("unknown MipmapFilter %q", string(text))
	}

	*f = val

	return nil
}

// ErrorMetrics describes error weighting for compression.
type ErrorMetrics int

// ErrorMetrics values mirror TexConvert.cfg errorMetrics names.
const (
	ErrorMetricsDefault  ErrorMetrics = iota // Default error metrics.
	ErrorMetricsDistance                     // Distance error metrics.
)

func (m ErrorMetrics) String() string {
	switch m {
	case ErrorMetricsDefault:
		return "Default"
	case ErrorMetricsDistance:
		return "Distance"
	default:
		return fmt.Sprintf("ErrorMetrics(%d)", int(m))
	}
}

// ParseErrorMetrics converts a config string to ErrorMetrics.
func ParseErrorMetrics(s string) (ErrorMetrics, bool) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.ToLower(s)

	if s == "" || s == "default" {
		return ErrorMetricsDefault, true
	}

	if s == "distance" {
		return ErrorMetricsDistance, true
	}

	return ErrorMetricsDefault, false
}

// MarshalText implements encoding.TextMarshaler.
func (m ErrorMetrics) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *ErrorMetrics) UnmarshalText(text []byte) error {
	val, ok := ParseErrorMetrics(string(text))
	if !ok {
		return fmt.Errorf("unknown ErrorMetrics %q", string(text))
	}
	*m = val
	return nil
}

// SwizzleSource indicates which input channel to use.
type SwizzleSource int

// SwizzleSource values represent input channels.
const (
	SwizzleA SwizzleSource = iota // A channel.
	SwizzleR                      // R channel.
	SwizzleG                      // G channel.
	SwizzleB                      // B channel.
)

// String returns the string representation of the SwizzleSource.
func (s SwizzleSource) String() string {
	switch s {
	case SwizzleA:
		return "A"
	case SwizzleR:
		return "R"
	case SwizzleG:
		return "G"
	case SwizzleB:
		return "B"
	default:
		return fmt.Sprintf("SwizzleSource(%d)", int(s))
	}
}

// parseSwizzleSource parses a swizzle source from a string.
func parseSwizzleSource(s string) (SwizzleSource, bool) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.ToUpper(s)

	switch s {
	case "A":
		return SwizzleA, true
	case "R":
		return SwizzleR, true
	case "G":
		return SwizzleG, true
	case "B":
		return SwizzleB, true
	default:
		return SwizzleA, false
	}
}

// SwizzleExpr describes a single channel expression (e.g. R, 1-A, 0, 1).
// Valid expressions are: A, R, G, B, 0, 1, 1-A, 1-R, 1-G, 1-B.
type SwizzleExpr struct {
	Source     SwizzleSource // The source channel.
	Valid      bool          // Whether the expression is valid.
	IsConst    bool          // Whether the expression is a constant.
	ConstValue byte          // The constant value.
	Invert     bool          // Whether the expression is inverted.
}

// ParseSwizzleExpr parses expressions like R, A, 0, 1, 1-R, 1-A.
func ParseSwizzleExpr(s string) (SwizzleExpr, error) {
	raw := strings.TrimSpace(s)
	raw = strings.Trim(raw, "\"")
	if raw == "" {
		return SwizzleExpr{}, errors.New("empty swizzle expression")
	}

	upper := strings.ToUpper(raw)
	if upper == "0" {
		return SwizzleExpr{Valid: true, IsConst: true, ConstValue: 0}, nil
	}

	if upper == "1" {
		return SwizzleExpr{Valid: true, IsConst: true, ConstValue: 255}, nil
	}

	if strings.HasPrefix(upper, "1-") {
		src, ok := parseSwizzleSource(strings.TrimPrefix(upper, "1-"))
		if !ok {
			return SwizzleExpr{}, fmt.Errorf("unknown swizzle source in %q", raw)
		}

		return SwizzleExpr{Valid: true, Source: src, Invert: true}, nil
	}

	src, ok := parseSwizzleSource(upper)
	if !ok {
		return SwizzleExpr{}, fmt.Errorf("unknown swizzle expression %q", raw)
	}

	return SwizzleExpr{Valid: true, Source: src}, nil
}

// String returns the string representation of the SwizzleExpr.
func (e SwizzleExpr) String() string {
	if !e.Valid {
		return ""
	}

	if e.IsConst {
		if e.ConstValue == 0 {
			return "0"
		}
		return "1"
	}

	src := e.Source.String()
	if e.Invert {
		return "1-" + src
	}

	return src
}

// MarshalText implements encoding.TextMarshaler.
func (e SwizzleExpr) MarshalText() ([]byte, error) {
	if !e.Valid {
		return nil, nil
	}

	return []byte(e.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (e *SwizzleExpr) UnmarshalText(text []byte) error {
	val, err := ParseSwizzleExpr(string(text))
	if err != nil {
		return err
	}

	*e = val
	return nil
}

// apply applies the swizzle expression to the given channels.
func (e SwizzleExpr) apply(r, g, b, a byte, def SwizzleSource) byte {
	if !e.Valid {
		return channelValue(def, r, g, b, a)
	}

	if e.IsConst {
		return e.ConstValue
	}

	v := channelValue(e.Source, r, g, b, a)
	if e.Invert {
		return 255 - v
	}

	return v
}

// isDefault checks if the swizzle expression is the default.
func (e SwizzleExpr) isDefault(def SwizzleSource) bool {
	if !e.Valid {
		return true
	}

	if e.IsConst {
		return false
	}

	return e.Source == def && !e.Invert
}

// channelValue returns the value of the given channel.
func channelValue(src SwizzleSource, r, g, b, a byte) byte {
	switch src {
	case SwizzleA:
		return a
	case SwizzleR:
		return r
	case SwizzleG:
		return g
	case SwizzleB:
		return b
	default:
		return 0
	}
}

// ChannelSwizzle maps output channels to input expressions.
type ChannelSwizzle struct {
	R SwizzleExpr `json:"r,omitempty" yaml:"r,omitempty"` // R channel expression.
	G SwizzleExpr `json:"g,omitempty" yaml:"g,omitempty"` // G channel expression.
	B SwizzleExpr `json:"b,omitempty" yaml:"b,omitempty"` // B channel expression.
	A SwizzleExpr `json:"a,omitempty" yaml:"a,omitempty"` // A channel expression.
}

// IsIdentity reports whether the swizzle is an identity.
func (s ChannelSwizzle) IsIdentity() bool {
	return s.R.isDefault(SwizzleR) && s.G.isDefault(SwizzleG) && s.B.isDefault(SwizzleB) && s.A.isDefault(SwizzleA)
}

// apply applies the channel swizzle to the given channels.
func (s ChannelSwizzle) apply(r, g, b, a byte) (byte, byte, byte, byte) {
	r2 := s.R.apply(r, g, b, a, SwizzleR)
	g2 := s.G.apply(r, g, b, a, SwizzleG)
	b2 := s.B.apply(r, g, b, a, SwizzleB)
	a2 := s.A.apply(r, g, b, a, SwizzleA)

	return r2, g2, b2, a2
}

// ZIWSTag returns the SWIZTAGG payload for this swizzle.
// The 4 bytes map to A, R, G, B swizzle selectors used by BI tools.
func (s ChannelSwizzle) ZIWSTag() ([4]byte, bool, error) {
	if s.IsIdentity() {
		return [4]byte{}, false, nil
	}

	var tag [4]byte
	v, ok := ziwsValue(s.A, SwizzleA)
	if !ok {
		return tag, false, fmt.Errorf("unsupported swizzle expr for A: %q", s.A.String())
	}

	tag[0] = v
	v, ok = ziwsValue(s.R, SwizzleR)
	if !ok {
		return tag, false, fmt.Errorf("unsupported swizzle expr for R: %q", s.R.String())
	}

	tag[1] = v
	v, ok = ziwsValue(s.G, SwizzleG)
	if !ok {
		return tag, false, fmt.Errorf("unsupported swizzle expr for G: %q", s.G.String())
	}

	tag[2] = v
	v, ok = ziwsValue(s.B, SwizzleB)
	if !ok {
		return tag, false, fmt.Errorf("unsupported swizzle expr for B: %q", s.B.String())
	}

	tag[3] = v
	return tag, true, nil
}

// ziwsValue returns the value of the given swizzle expression.
const (
	ziwsA    = 0x00
	ziwsR    = 0x01
	ziwsG    = 0x02
	ziwsB    = 0x03
	ziwsInvA = 0x04
	ziwsInvR = 0x05
	ziwsInvG = 0x06
	ziwsInvB = 0x07
	ziwsOne  = 0x08
	ziwsZero = 0x09
)

// ziwsValue returns the value of the given swizzle expression.
func ziwsValue(expr SwizzleExpr, def SwizzleSource) (byte, bool) {
	if !expr.Valid {
		return ziwsValue(SwizzleExpr{Valid: true, Source: def}, def)
	}

	if expr.IsConst {
		if expr.ConstValue == 0 {
			return ziwsZero, true
		}
		return ziwsOne, true
	}

	src := expr.Source

	// Return the value of the given swizzle expression.
	switch src {
	case SwizzleA:
		if expr.Invert {
			return ziwsInvA, true
		}
		return ziwsA, true

	case SwizzleR:
		if expr.Invert {
			return ziwsInvR, true
		}
		return ziwsR, true

	case SwizzleG:
		if expr.Invert {
			return ziwsInvG, true
		}
		return ziwsG, true

	case SwizzleB:
		if expr.Invert {
			return ziwsInvB, true
		}
		return ziwsB, true

	default:
		return 0, false
	}
}
