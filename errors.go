package paa

import "errors"

// PAA decode/encode errors. Use errors.Is to check.
var (
	// ErrNoMipmaps is returned when no mipmaps are found in the PAA file.
	ErrNoMipmaps = errors.New("paa: no mipmaps found")
	// ErrInvalidMagic is returned when the magic/format is invalid.
	ErrInvalidMagic = errors.New("paa: invalid magic/format")
	// ErrMissingSFFO is returned when the SFFO tag is missing.
	ErrMissingSFFO = errors.New("paa: missing SFFO tag")
	// ErrInsufficientData is returned when there is not enough data for ARGB8.
	ErrInsufficientData = errors.New("paa: not enough data for ARGB8")
	// ErrUnsupportedPixelFmt is returned when the pixel format is unsupported for decode.
	ErrUnsupportedPixelFmt = errors.New("paa: unsupported pixel format for decode")
	// ErrUnsupportedFormat is returned when the format is unsupported for conversion.
	ErrUnsupportedFormat = errors.New("paa: unsupported format for conversion")
	// ErrLZODecompress is returned when LZO decompression failed.
	ErrLZODecompress = errors.New("paa: LZO decompression failed")
	// ErrLZSSDecompress is returned when LZSS decompression failed.
	ErrLZSSDecompress = errors.New("paa: LZSS decompression failed")
	// ErrDXTDecode is returned when DXT decode failed.
	ErrDXTDecode = errors.New("paa: DXT decode failed")
	// ErrInvalidDimensions is returned when the dimensions exceed the PAA uint16 range (0-65535).
	ErrInvalidDimensions = errors.New("paa: dimensions exceed PAA uint16 range (0-65535)")
)
