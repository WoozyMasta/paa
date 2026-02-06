package paa

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

// Decode reads a PAA stream and returns the first mip level as an image.
// It implements the signature required by image.RegisterFormat.
func Decode(r io.Reader) (image.Image, error) {
	p, err := DecodePAA(r)
	if err != nil {
		return nil, err
	}

	if len(p.MipMaps) == 0 {
		return nil, ErrNoMipmaps
	}

	img, err := p.MipMaps[0].Image()
	if err != nil {
		return nil, err
	}

	return applySwizzleTag(p, img), nil
}

// DecodeConfig reads only the dimensions of the first mip level.
// It implements the signature required by image.RegisterFormat.
func DecodeConfig(r io.Reader) (image.Config, error) {
	p, err := DecodePAA(r)
	if err != nil {
		return image.Config{}, err
	}

	if len(p.MipMaps) == 0 {
		return image.Config{}, ErrNoMipmaps
	}

	mm := p.MipMaps[0]
	return image.Config{
		ColorModel: color.NRGBAModel,
		Width:      int(mm.Width),
		Height:     int(mm.Height),
	}, nil
}

// applySwizzleTag applies the ZIWS tag to the image if it exists and the texture is a DXT5 normal map.
func applySwizzleTag(p *PAA, img image.Image) image.Image {
	tag, ok := p.Taggs["ZIWS"]
	if !ok || len(tag) != 4 || p.Type != PaxDXT5 {
		return img
	}

	var swiz [4]byte
	copy(swiz[:], tag)
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
// The SFFO tag holds a table of absolute file offsets to each mip level; we seek
// to each offset and read the mip (width, height, length, payload). If r does
// not implement io.Seeker, the entire stream is read into memory first so that
// we can seek (e.g. for image.Decode which may pass a non-seekable reader).
func DecodePAA(r io.Reader) (*PAA, error) {
	seeker, ok := r.(io.Seeker)
	if !ok {
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}

		br := bytes.NewReader(data)
		r = br
		seeker = br
	}

	// Read PaxType as first tag.
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	pType, ok := PaxTypeFromBytes(header)
	if !ok {
		return nil, ErrInvalidMagic
	}

	paa := &PAA{
		Type:  pType,
		Taggs: make(map[string][]byte),
	}

	// Read GGAT tags.
	for {
		sig := make([]byte, 4)
		if _, err := io.ReadFull(r, sig); err != nil {
			return nil, err
		}

		if string(sig) != "GGAT" {
			break
		}

		nameBuf := make([]byte, 4)
		if _, err := io.ReadFull(r, nameBuf); err != nil {
			return nil, err
		}
		name := string(nameBuf)

		var size uint32
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			return nil, err
		}

		data := make([]byte, size)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
		paa.Taggs[name] = data
	}

	// Read SFFO tag.
	sffoData, ok := paa.Taggs["SFFO"]
	if !ok {
		return nil, ErrMissingSFFO
	}

	// Read mipmaps from SFFO offsets.
	for i := 0; i < len(sffoData)/4; i++ {
		offset := binary.LittleEndian.Uint32(sffoData[i*4 : i*4+4])
		if offset == 0 {
			continue
		}

		if _, err := seeker.Seek(int64(offset), io.SeekStart); err != nil {
			return nil, err
		}

		mm, err := readMipMap(r, paa.Type)
		if err != nil {
			return nil, err
		}

		if mm == nil {
			continue
		}

		paa.MipMaps = append(paa.MipMaps, mm)
	}

	return paa, nil
}
