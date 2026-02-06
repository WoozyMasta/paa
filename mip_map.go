package paa

import (
	"encoding/binary"
	"errors"
	"image"
	"io"

	"github.com/woozymasta/bcn"
	"github.com/woozymasta/lzo"
	"github.com/woozymasta/lzss"
)

// MipMap holds one mip level: dimensions and raw decoded pixel/block data.
type MipMap struct {
	Data   []byte  // Raw decoded pixel/block data.
	Type   PaxType // Type is the PaxType of the mipmap.
	Width  uint16  // Width is the width of the mipmap.
	Height uint16  // Height is the height of the mipmap.
}

// readMipMap reads one mipmap from r at the current position.
// Format: width (2), height (2), size (3 bytes LE), data (size bytes).
// For Arma2+ DXT, the top bit of width indicates LZO compression; it is masked for dimensions.
// Returns (nil, nil) for the dummy mip (width==0 && height==0).
func readMipMap(r io.Reader, paxType PaxType) (*MipMap, error) {
	var w, h uint16
	if err := binary.Read(r, binary.LittleEndian, &w); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		return nil, err
	}
	if w == 0 && h == 0 {
		return nil, nil
	}

	// Arma2+ LZO: for DXT only, top bit of width is compression flag.
	lzoFlag := isDXT(paxType) && (w&0x8000) != 0
	width := w
	if lzoFlag {
		width = w & 0x7FFF
	}
	height := h

	var sizeBuf [3]byte
	if _, err := io.ReadFull(r, sizeBuf[:]); err != nil {
		return nil, err
	}
	storedSize := int(sizeBuf[0]) | int(sizeBuf[1])<<8 | int(sizeBuf[2])<<16

	payload := make([]byte, storedSize)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	expectedRaw := expectedMipSize(paxType, int(width), int(height))
	if expectedRaw < 0 {
		return nil, ErrUnsupportedPixelFmt
	}

	var raw []byte
	if storedSize == expectedRaw {
		raw = payload
	} else if isDXT(paxType) {
		// DXT: only LZO when top bit of width is set.
		if !lzoFlag {
			return nil, ErrInsufficientData
		}
		dec, err := lzo.Decompress(payload, lzo.DefaultDecompressOptions(expectedRaw))
		if err != nil {
			if errors.Is(err, lzo.ErrLookBehindUnderrun) || errors.Is(err, lzo.ErrInputOverrun) {
				return nil, errors.Join(ErrLZODecompress, err)
			}
			return nil, errors.Join(ErrLZODecompress, err)
		}
		raw = dec
	} else {
		// Non-DXT: LZSS (signed checksum, lenient).
		dec, err := lzss.Decompress(payload, expectedRaw, lzss.SignedLenientOptions())
		if err != nil {
			return nil, errors.Join(ErrLZSSDecompress, err)
		}
		raw = dec
	}

	return &MipMap{
		Width:  width,
		Height: height,
		Data:   raw,
		Type:   paxType,
	}, nil
}

// expectedMipSize returns the expected raw byte size for this mip (decoded/encoded size).
// Returns -1 for unsupported format.
func expectedMipSize(paxType PaxType, width, height int) int {
	if width <= 0 || height <= 0 {
		return 0
	}
	switch paxType {
	case PaxDXT1:
		return ((width + 3) / 4) * ((height + 3) / 4) * 8
	case PaxDXT2, PaxDXT3, PaxDXT4, PaxDXT5:
		return ((width + 3) / 4) * ((height + 3) / 4) * 16
	case PaxARGB8:
		return width * height * 4
	case PaxARGBA5, PaxARGB4:
		return width * height * 2
	case PaxGRAYA:
		return width * height * 2
	default:
		return -1
	}
}

// Image decodes the mipmap into an image.Image (NRGBA for non-DXT, DXT decoded via bcn).
func (m *MipMap) Image() (image.Image, error) {
	w, h := int(m.Width), int(m.Height)
	if w <= 0 || h <= 0 || len(m.Data) == 0 {
		return nil, ErrInsufficientData
	}

	if isDXT(m.Type) {
		bf := paxToBcnFormat(m.Type)
		if bf == bcn.FormatUnknown {
			return nil, ErrUnsupportedPixelFmt
		}
		img, err := bcn.DecodeImage(m.Data, w, h, bf)
		if err != nil {
			return nil, errors.Join(ErrDXTDecode, err)
		}
		return img, nil
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if err := decodePixelFormat(m.Type, m.Data, w, h, img); err != nil {
		return nil, err
	}

	return img, nil
}

func paxToBcnFormat(p PaxType) bcn.Format {
	switch p {
	case PaxDXT1:
		return bcn.FormatDXT1
	case PaxDXT3:
		return bcn.FormatDXT3
	case PaxDXT5:
		return bcn.FormatDXT5
	case PaxDXT2:
		return bcn.FormatDXT3
	case PaxDXT4:
		return bcn.FormatDXT5
	default:
		return bcn.FormatUnknown
	}
}
