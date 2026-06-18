// SPDX-License-Identifier: MIT
// Copyright (c) 2026 WoozyMasta
// Source: github.com/woozymasta/paa

package paa

import (
	"image"
	"io"
)

// Encoder encodes images to PAA while reusing its internal buffers across calls,
// which avoids most per-image allocations in batch pipelines
// (e.g. converting many TGA/PNG files).
// Buffers grow to fit the largest image seen and are reused for smaller ones,
// so the steady-state allocation count drops to near zero.
//
// An Encoder is NOT safe for concurrent use; create one per worker goroutine
// (combine with BCn Workers=1 for the best batch throughput).
// The zero value is ready to use; NewEncoder is provided for clarity.
type Encoder struct {
	mipPool   []*image.NRGBA // reusable mip-chain buffers (bcn.GenerateMipmapsInto)
	mipImages []image.Image  // reusable view of the selected mip levels
	dxtBuf    []byte         // transient per-mip BCn output, reused each mip
	payloads  [][]byte       // per-mip retained payloads (LZO/raw), reused across calls
	mips      []mipBlock     // reusable per-mip metadata
}

// NewEncoder returns a ready-to-use Encoder.
func NewEncoder() *Encoder {
	return &Encoder{}
}

// Encode writes img as PAA with a single mip level using default settings.
// See the package-level Encode for details.
func (e *Encoder) Encode(w io.Writer, img image.Image) error {
	return e.encodeWithOptions(w, img, nil, nil)
}

// EncodeWithOptions writes img as PAA using opts.
// See the package-level EncodeWithOptions for details.
func (e *Encoder) EncodeWithOptions(w io.Writer, img image.Image, opts *EncodeOptions) error {
	return e.encodeWithOptions(w, img, opts, nil)
}

// EncodeWithOptionsAndMetadataHeaders writes img as PAA
// and returns compact metadata headers collected during the same encode pass.
func (e *Encoder) EncodeWithOptionsAndMetadataHeaders(w io.Writer, img image.Image, opts *EncodeOptions) (*MetadataHeaders, error) {
	meta := &MetadataHeaders{}
	if err := e.encodeWithOptions(w, img, opts, meta); err != nil {
		return nil, err
	}

	return meta, nil
}

// mipBlock holds one encoded mip level's payload and dimensions.
type mipBlock struct {
	data  []byte
	w, h  int
	useLZ bool
}

// payload returns the reusable retained-payload buffer for mip level i,
// growing the backing slice as needed.
func (e *Encoder) payload(i int) []byte {
	for i >= len(e.payloads) {
		e.payloads = append(e.payloads, nil)
	}

	return e.payloads[i]
}
