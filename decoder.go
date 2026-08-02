// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"

	"github.com/woozymasta/bcn"
	"github.com/woozymasta/lzo"
	"github.com/woozymasta/lzss"
)

// Decoder decodes the first mip level of PAA streams while reusing internal buffers across calls,
// which avoids most per-image allocations in batch pipelines
// (e.g. converting many PAA files to TGA/PNG).
// Buffers grow to fit the largest image seen and are reused for smaller ones.
//
// A Decoder is NOT safe for concurrent use; create one per worker goroutine.
// IMPORTANT: the returned image shares the Decoder's reusable pixel buffer
// and is only valid until the next Decode call on the same Decoder.
// Copy it if you need to retain it across calls.
type Decoder struct {
	img        *image.NRGBA // reusable decoded output
	payload    []byte       // reusable compressed mip payload
	raw        []byte       // reusable decompressed DXT/pixel buffer
	sffo       []byte       // reusable SFFO tag payload buffer
	mipHeaders []MipHeader  // reusable mip header slice (DecodeMetadataHeaders)
}

// NewDecoder returns a ready-to-use Decoder.
func NewDecoder() *Decoder {
	return &Decoder{}
}

// Decode reads a PAA stream and returns its first mip level as an image,
// reusing the Decoder's buffers. See the Decoder type docs for reuse semantics.
func (d *Decoder) Decode(r io.Reader) (image.Image, error) {
	return d.DecodeWithOptions(r, nil)
}

// DecodeWithOptions reads a PAA stream and returns its first mip level
// as an image using optional BCn decode settings, reusing the Decoder's buffers.
func (d *Decoder) DecodeWithOptions(r io.Reader, opts *DecodeOptions) (image.Image, error) {
	r, seeker, err := ensureSeeker(r)
	if err != nil {
		return nil, err
	}

	pType, err := readPaxType(r)
	if err != nil {
		return nil, err
	}

	tags, err := readFirstMipTags(r)
	if err != nil {
		return nil, err
	}

	for _, offset := range tags.offsets[:tags.offsetCount] {
		if _, err := seeker.Seek(int64(offset), io.SeekStart); err != nil {
			return nil, err
		}

		img, ok, err := d.readMipImage(r, pType, opts)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		return applyFirstMipSwizzle(tags, pType, img), nil
	}

	return nil, ErrNoMipmaps
}

// readMipImage reads one mip at the current position
// and decodes it into the Decoder's reusable image.
// ok is false for the dummy (0x0) mip.
func (d *Decoder) readMipImage(r io.Reader, paxType PaxType, opts *DecodeOptions) (*image.NRGBA, bool, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, false, err
	}
	w := binary.LittleEndian.Uint16(hdr[0:2])
	h := binary.LittleEndian.Uint16(hdr[2:4])
	if w == 0 && h == 0 {
		return nil, false, nil
	}

	// Arma2+ LZO: for DXT only, the top bit of width is the compression flag.
	lzoFlag := isDXT(paxType) && (w&0x8000) != 0
	width := int(w)
	if lzoFlag {
		width = int(w & 0x7FFF)
	}
	height := int(h)

	if err := checkMipBudget(width, height); err != nil {
		return nil, false, err
	}

	var sizeBuf [3]byte
	if _, err := io.ReadFull(r, sizeBuf[:]); err != nil {
		return nil, false, err
	}
	storedSize := int(sizeBuf[0]) | int(sizeBuf[1])<<8 | int(sizeBuf[2])<<16

	// Reject a payload larger than the remaining stream before allocating.
	if rem, ok := remainingBytes(r); ok && int64(storedSize) > rem {
		return nil, false, errors.Join(ErrInsufficientData, fmt.Errorf("mip payload %d, %d remain", storedSize, rem))
	}

	d.payload = ensureLen(d.payload, storedSize)
	if _, err := io.ReadFull(r, d.payload); err != nil {
		return nil, false, err
	}

	expectedRaw := expectedMipSize(paxType, width, height)
	if expectedRaw < 0 {
		return nil, false, ErrUnsupportedPixelFmt
	}

	var raw []byte
	switch {
	case storedSize == expectedRaw:
		raw = d.payload
	case isDXT(paxType):
		if !lzoFlag {
			return nil, false, ErrInsufficientData
		}

		d.raw = ensureLen(d.raw, expectedRaw)
		dec, nRead, derr := lzo.DecompressNInto(d.payload, d.raw)
		if derr != nil {
			return nil, false, errors.Join(ErrLZODecompress, derr)
		}
		if nRead != len(d.payload) {
			return nil, false, errors.Join(ErrLZODecompress, fmt.Errorf("%w: consumed %d of %d bytes", ErrLZOTrailingInput, nRead, len(d.payload)))
		}
		raw = dec
	default:
		// Non-DXT LZSS (rare): allocates per call.
		dec, derr := lzss.Decompress(d.payload, expectedRaw, lzss.SignedLenientOptions())
		if derr != nil {
			return nil, false, errors.Join(ErrLZSSDecompress, derr)
		}
		raw = dec
	}

	if isDXT(paxType) {
		bf := paxToBcnFormat(paxType)
		if bf == bcn.FormatUnknown {
			return nil, false, ErrUnsupportedPixelFmt
		}

		bcnOpts := resolveDecodeBCnOptions(opts)

		img, derr := bcn.DecodeImageInto(d.img, raw, width, height, bf, bcnOpts)
		if derr != nil {
			return nil, false, errors.Join(ErrDXTDecode, derr)
		}
		d.img = img

		return img, true, nil
	}

	d.img = reuseNRGBA(d.img, width, height)
	if err := decodePixelFormat(paxType, raw, width, height, d.img); err != nil {
		return nil, false, err
	}

	return d.img, true, nil
}

// ensureLen returns b resliced to length n, growing (reallocating) when needed.
func ensureLen(b []byte, n int) []byte {
	if cap(b) < n {
		return make([]byte, n)
	}

	return b[:n]
}

// reuseNRGBA returns an NRGBA of width x height,
// reusing img's Pix buffer when its capacity is large enough,
// otherwise allocating a new image.
func reuseNRGBA(img *image.NRGBA, width, height int) *image.NRGBA {
	need := width * height * 4
	if img != nil && cap(img.Pix) >= need {
		return &image.NRGBA{Pix: img.Pix[:need], Stride: width * 4, Rect: image.Rect(0, 0, width, height)}
	}

	return image.NewNRGBA(image.Rect(0, 0, width, height))
}
