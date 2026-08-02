// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/woozymasta/paa"
	"github.com/woozymasta/paa/texconfig"
)

func Example() {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	img.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 128, B: 64, A: 255})

	var encoded bytes.Buffer
	if err := paa.Encode(&encoded, img); err != nil {
		panic(err)
	}

	decoded, err := paa.Decode(&encoded)
	if err != nil {
		panic(err)
	}

	bounds := decoded.Bounds()
	fmt.Printf("decoded texture: %dx%d\n", bounds.Dx(), bounds.Dy())

	// Output:
	// decoded texture: 4x4
}

func ExampleEncodeWithTexConfig() {
	cfg, err := texconfig.DefaultTexConvertConfig()
	if err != nil {
		panic(err)
	}

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	img.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 128, B: 64, A: 255})

	var encoded bytes.Buffer
	if err := paa.EncodeWithTexConfig(&encoded, img, "vehicle_co.paa", cfg); err != nil {
		panic(err)
	}

	fmt.Println(encoded.Len() > 0)

	// Output:
	// true
}

func ExampleDecodeToDDS() {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	noMips := false

	var encoded bytes.Buffer
	if err := paa.EncodeWithOptions(&encoded, img, &paa.EncodeOptions{
		Type:            paa.PaxARGB8,
		GenerateMipmaps: &noMips,
	}); err != nil {
		panic(err)
	}

	dds, err := paa.DecodeToDDS(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s %dx%d\n", dds.Format, dds.Width, dds.Height)

	// Output:
	// BGRA8 4x4
}

func ExampleEncoder_Encode() {
	encoder := paa.NewEncoder()
	for _, size := range []int{4, 8} {
		img := image.NewNRGBA(image.Rect(0, 0, size, size))
		var encoded bytes.Buffer
		if err := encoder.Encode(&encoded, img); err != nil {
			panic(err)
		}

		fmt.Println(encoded.Len() > 0)
	}

	// Output:
	// true
	// true
}

func ExampleDecodeMetadataHeaders() {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	var encoded bytes.Buffer
	if err := paa.Encode(&encoded, img); err != nil {
		panic(err)
	}

	headers, err := paa.DecodeMetadataHeaders(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		panic(err)
	}

	base := headers.MipHeaders[0]
	fmt.Printf("base mip: %dx%d\n", base.Width, base.Height)

	// Output:
	// base mip: 16x8
}
