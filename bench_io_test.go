package paa

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/woozymasta/bcn"
)

const (
	benchMainFlowWidth  = 1024
	benchMainFlowHeight = 1024
)

type mainFlowBenchFixture struct {
	img        *image.NRGBA
	encodedPAA []byte
	encodeOpts *EncodeOptions
}

var (
	mainFlowFixture    *mainFlowBenchFixture
	mainFlowFixtureErr error

	mainFlowSinkImage           image.Image
	mainFlowSinkSize            int
	mainFlowSinkMetadata        *Metadata
	mainFlowSinkMetadataHeaders *MetadataHeaders
)

func buildMainFlowBenchFixture() (*mainFlowBenchFixture, error) {
	img := image.NewNRGBA(image.Rect(0, 0, benchMainFlowWidth, benchMainFlowHeight))
	for y := range benchMainFlowHeight {
		for x := range benchMainFlowWidth {
			r := uint8((x * 255) / (benchMainFlowWidth - 1))
			g := uint8((y * 255) / (benchMainFlowHeight - 1))
			b := uint8(((x + y) * 255) / (benchMainFlowWidth + benchMainFlowHeight - 2))
			a := uint8(((x ^ y) & 0xFF))
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}

	noMips := false
	opts := &EncodeOptions{
		Type:            PaxDXT5,
		GenerateMipmaps: &noMips,
		UseLZO:          true,
		BCn: &bcn.EncodeOptions{
			QualityLevel: bcn.QualityLevelFast,
		},
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, opts); err != nil {
		return nil, err
	}

	return &mainFlowBenchFixture{
		img:        img,
		encodedPAA: append([]byte(nil), buf.Bytes()...),
		encodeOpts: opts,
	}, nil
}

func requireMainFlowBenchFixture(b *testing.B) *mainFlowBenchFixture {
	b.Helper()

	if mainFlowFixture == nil && mainFlowFixtureErr == nil {
		mainFlowFixture, mainFlowFixtureErr = buildMainFlowBenchFixture()
	}

	if mainFlowFixtureErr != nil {
		b.Fatalf("build benchmark fixture: %v", mainFlowFixtureErr)
	}

	return mainFlowFixture
}

func BenchmarkMainFlowEncode(b *testing.B) {
	f := requireMainFlowBenchFixture(b)
	var buf bytes.Buffer
	buf.Grow(len(f.encodedPAA))

	b.ReportAllocs()
	b.SetBytes(int64(len(f.img.Pix)))
	b.ResetTimer()

	for range b.N {
		buf.Reset()
		if err := EncodeWithOptions(&buf, f.img, f.encodeOpts); err != nil {
			b.Fatalf("encode: %v", err)
		}

		mainFlowSinkSize = buf.Len()
	}
}

func BenchmarkMainFlowDecode(b *testing.B) {
	f := requireMainFlowBenchFixture(b)

	b.ReportAllocs()
	b.SetBytes(int64(len(f.encodedPAA)))
	b.ResetTimer()

	for range b.N {
		img, err := Decode(bytes.NewReader(f.encodedPAA))
		if err != nil {
			b.Fatalf("decode: %v", err)
		}

		mainFlowSinkImage = img
	}
}

func BenchmarkMainFlowRoundTrip(b *testing.B) {
	f := requireMainFlowBenchFixture(b)
	var buf bytes.Buffer
	buf.Grow(len(f.encodedPAA))

	b.ReportAllocs()
	b.SetBytes(int64(len(f.img.Pix)))
	b.ResetTimer()

	for range b.N {
		buf.Reset()
		if err := EncodeWithOptions(&buf, f.img, f.encodeOpts); err != nil {
			b.Fatalf("encode: %v", err)
		}

		img, err := Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			b.Fatalf("decode: %v", err)
		}

		mainFlowSinkImage = img
		mainFlowSinkSize = buf.Len()
	}
}

func BenchmarkMainFlowMetadataDecode(b *testing.B) {
	f := requireMainFlowBenchFixture(b)

	b.Run("Full", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(f.encodedPAA)))
		b.ResetTimer()

		for range b.N {
			meta, err := DecodeMetadata(bytes.NewReader(f.encodedPAA))
			if err != nil {
				b.Fatalf("decode metadata: %v", err)
			}

			mainFlowSinkMetadata = meta
		}
	})

	b.Run("Headers", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(f.encodedPAA)))
		b.ResetTimer()

		for range b.N {
			meta, err := DecodeMetadataHeaders(bytes.NewReader(f.encodedPAA))
			if err != nil {
				b.Fatalf("decode metadata headers: %v", err)
			}

			mainFlowSinkMetadataHeaders = meta
		}
	})

	b.Run("FullBytes", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(f.encodedPAA)))
		b.ResetTimer()

		for range b.N {
			meta, err := DecodeMetadataBytes(f.encodedPAA)
			if err != nil {
				b.Fatalf("decode metadata bytes: %v", err)
			}

			mainFlowSinkMetadata = meta
		}
	})

	b.Run("HeadersBytes", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(f.encodedPAA)))
		b.ResetTimer()

		for range b.N {
			meta, err := DecodeMetadataHeadersBytes(f.encodedPAA)
			if err != nil {
				b.Fatalf("decode metadata headers bytes: %v", err)
			}

			mainFlowSinkMetadataHeaders = meta
		}
	})
}
