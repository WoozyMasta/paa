// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

// Decode reads a PAA stream and returns the first mip level as an image.
// It implements the signature required by image.RegisterFormat.
func Decode(r io.Reader) (image.Image, error) {
	return DecodeWithOptions(r, nil)
}

// DecodeWithOptions reads a PAA stream and returns the first mip level as an image
// using optional BCn decode settings.
func DecodeWithOptions(r io.Reader, opts *DecodeOptions) (image.Image, error) {
	pType, tags, mm, err := readFirstMipMap(r)
	if err != nil {
		return nil, err
	}

	img, err := mm.ImageWithOptions(opts)
	if err != nil {
		return nil, err
	}

	return applyFirstMipSwizzle(tags, pType, img), nil
}

// DecodeConfig reads only the dimensions of the first mip level.
// It implements the signature required by image.RegisterFormat.
func DecodeConfig(r io.Reader) (image.Config, error) {
	width, height, err := readFirstMipHeader(r)
	if err != nil {
		return image.Config{}, err
	}

	return image.Config{
		ColorModel: color.NRGBAModel,
		Width:      int(width),  //nolint:gosec // mip dimensions are uint16 by format.
		Height:     int(height), //nolint:gosec // mip dimensions are uint16 by format.
	}, nil
}

// readFirstMipMap reads headers/tags and returns the first non-dummy mip map only.
func readFirstMipMap(r io.Reader) (PaxType, firstMipTags, *MipMap, error) {
	var err error
	r, seeker, err := ensureSeeker(r)
	if err != nil {
		return 0, firstMipTags{}, nil, err
	}

	pType, err := readPaxType(r)
	if err != nil {
		return 0, firstMipTags{}, nil, err
	}

	tags, err := readFirstMipTags(r)
	if err != nil {
		return 0, firstMipTags{}, nil, err
	}

	for _, offset := range tags.offsets[:tags.offsetCount] {
		if _, err := seeker.Seek(int64(offset), io.SeekStart); err != nil {
			return 0, firstMipTags{}, nil, err
		}

		mm, err := readMipMap(r, pType)
		if err != nil {
			return 0, firstMipTags{}, nil, err
		}

		if mm == nil {
			continue
		}

		return pType, tags, mm, nil
	}

	return 0, firstMipTags{}, nil, ErrNoMipmaps
}

// readFirstMipHeader reads only width/height for the first non-dummy mip map.
func readFirstMipHeader(r io.Reader) (uint16, uint16, error) {
	var err error
	r, seeker, err := ensureSeeker(r)
	if err != nil {
		return 0, 0, err
	}

	pType, err := readPaxType(r)
	if err != nil {
		return 0, 0, err
	}

	tags, err := readFirstMipTags(r)
	if err != nil {
		return 0, 0, err
	}

	for _, offset := range tags.offsets[:tags.offsetCount] {
		if _, err := seeker.Seek(int64(offset), io.SeekStart); err != nil {
			return 0, 0, err
		}

		var w, h uint16
		if err := binary.Read(r, binary.LittleEndian, &w); err != nil {
			return 0, 0, err
		}
		if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
			return 0, 0, err
		}

		if w == 0 && h == 0 {
			continue
		}

		if isDXT(pType) && (w&0x8000) != 0 {
			w &= 0x7FFF
		}

		return w, h, nil
	}

	return 0, 0, ErrNoMipmaps
}

// applySwizzleTag applies ZIWS payload on decoded image for DXT5 normal maps.
func applySwizzleTag(tags map[string][]byte, pType PaxType, img image.Image) image.Image {
	tag, ok := tags["ZIWS"]
	if !ok || len(tag) != 4 || pType != PaxDXT5 {
		return img
	}

	var swiz [4]byte
	copy(swiz[:], tag)
	return applySwizzlePayloadTag(swiz, pType, img)
}

// applyFirstMipSwizzle applies the optional ZIWS payload retained by the fast parser.
func applyFirstMipSwizzle(tags firstMipTags, pType PaxType, img image.Image) image.Image {
	if !tags.hasSwizzle || pType != PaxDXT5 {
		return img
	}

	return applySwizzlePayloadTag(tags.swizzle, pType, img)
}

// applySwizzlePayloadTag applies a validated ZIWS payload to a DXT5 image.
func applySwizzlePayloadTag(swiz [4]byte, pType PaxType, img image.Image) image.Image {
	if pType != PaxDXT5 {
		return img
	}
	if swiz == swizzleDXT5NM {
		return unswizzleNormalMap(img)
	}

	if swiz == [4]byte{0x02, 0x09, 0x03, 0x09} {
		return applyADSHQSwizzle(img)
	}

	return applySwizzlePayload(img, swiz)
}

// DecodePAA reads a full PAA structure from the stream.
//
// File layout: 2-byte magic (PaxType), then GGAT tags (name + size + payload).
// The SFFO tag holds a table of absolute file offsets to each mip level;
// we seek to each offset and read the mip (width, height, length, payload).
// If r does not implement io.Seeker, the entire stream is read into memory first
// so that we can seek (e.g. for image.Decode which may pass a non-seekable reader).
func DecodePAA(r io.Reader) (*PAA, error) {
	var err error
	r, seeker, err := ensureSeeker(r)
	if err != nil {
		return nil, err
	}

	pType, err := readPaxType(r)
	if err != nil {
		return nil, err
	}

	tags, err := readGGATTags(r)
	if err != nil {
		return nil, err
	}

	offsets, err := sffoOffsets(tags)
	if err != nil {
		return nil, err
	}

	paa := &PAA{
		Type:  pType,
		Taggs: tags,
	}

	// Read mipmaps from SFFO offsets while bounding the aggregate decoded payload.
	remainingDecoded := maxDecodedMipBytes
	for _, offset := range offsets {
		if _, err := seeker.Seek(int64(offset), io.SeekStart); err != nil {
			return nil, err
		}

		mm, err := readMipMapWithLimit(r, paa.Type, remainingDecoded)
		if err != nil {
			return nil, err
		}

		if mm == nil {
			continue
		}
		remainingDecoded -= len(mm.Data)

		paa.MipMaps = append(paa.MipMaps, mm)
	}

	return paa, nil
}

// DecodeNthMip decodes the nth mip level (0-based) from a PAA stream.
// Returns ErrNoMipmaps if n is out of range.
func DecodeNthMip(r io.Reader, n int, opts *DecodeOptions) (image.Image, error) {
	var err error
	r, seeker, err := ensureSeeker(r)
	if err != nil {
		return nil, err
	}

	pType, err := readPaxType(r)
	if err != nil {
		return nil, err
	}

	tags, err := readGGATTags(r)
	if err != nil {
		return nil, err
	}

	offsets, err := sffoOffsets(tags)
	if err != nil {
		return nil, err
	}

	if n < 0 || n >= len(offsets) {
		return nil, fmt.Errorf("%w: mip index %d out of range [0, %d)", ErrNoMipmaps, n, len(offsets))
	}

	if _, err := seeker.Seek(int64(offsets[n]), io.SeekStart); err != nil {
		return nil, err
	}

	mm, err := readMipMap(r, pType)
	if err != nil {
		return nil, err
	}
	if mm == nil {
		return nil, ErrNoMipmaps
	}

	img, err := mm.ImageWithOptions(opts)
	if err != nil {
		return nil, err
	}

	return applySwizzleTag(tags, pType, img), nil
}

// AllImages decodes all mip levels and returns them as a slice.
// Swizzle tags (ZIWS) are applied to each level, consistent with Decode.
func (p *PAA) AllImages(opts *DecodeOptions) ([]image.Image, error) {
	if p == nil {
		return nil, ErrNilPAA
	}
	if len(p.MipMaps) == 0 {
		return nil, ErrNoMipmaps
	}

	imgs := make([]image.Image, 0, len(p.MipMaps))
	for _, mm := range p.MipMaps {
		if mm == nil {
			continue
		}

		img, err := mm.ImageWithOptions(opts)
		if err != nil {
			return nil, err
		}

		imgs = append(imgs, applySwizzleTag(p.Taggs, p.Type, img))
	}

	if len(imgs) == 0 {
		return nil, ErrNoMipmaps
	}

	return imgs, nil
}
