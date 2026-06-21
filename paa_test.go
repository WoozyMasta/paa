package paa

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/woozymasta/bcn"
	"github.com/woozymasta/paa/texconfig"
)

func parseTagg(data []byte) map[string][]byte {
	taggs := make(map[string][]byte)
	i := 2 // skip PaxType
	for i+8 <= len(data) {
		if string(data[i:i+4]) != "GGAT" {
			break
		}
		name := string(data[i+4 : i+8])
		size := int(uint32(data[i+8]) | uint32(data[i+9])<<8 | uint32(data[i+10])<<16 | uint32(data[i+11])<<24)
		if i+12+size > len(data) {
			break
		}
		taggs[name] = data[i+12 : i+12+size]
		i += 12 + size
	}
	return taggs
}

func firstMipOffset(taggs map[string][]byte) int {
	sffo, ok := taggs["SFFO"]
	if !ok || len(sffo) < 4 {
		return 0
	}
	return int(uint32(sffo[0]) | uint32(sffo[1])<<8 | uint32(sffo[2])<<16 | uint32(sffo[3])<<24)
}

func countNonZeroOffsets(taggs map[string][]byte) int {
	sffo, ok := taggs["SFFO"]
	if !ok {
		return 0
	}
	count := 0
	for i := 0; i+4 <= len(sffo); i += 4 {
		v := uint32(sffo[i]) | uint32(sffo[i+1])<<8 | uint32(sffo[i+2])<<16 | uint32(sffo[i+3])<<24
		if v != 0 {
			count++
		}
	}
	return count
}

func TestRoundTrip(t *testing.T) {
	// Create a small RGBA image (8x8), encode to PAA, decode back, check dimensions.
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 32), G: uint8(y * 32), B: 128, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	decoded, err := Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded.Bounds().Dx() != 8 || decoded.Bounds().Dy() != 8 {
		t.Errorf("decoded size = %dx%d, want 8x8", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func TestDecodeConfig(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 255
	}

	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	cfg, err := DecodeConfig(&buf)
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}
	if cfg.Width != 4 || cfg.Height != 4 {
		t.Errorf("DecodeConfig = %dx%d, want 4x4", cfg.Width, cfg.Height)
	}
}

func TestDecodeAndConfigUseFirstMipOnly(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, nil); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}

	data := buf.Bytes()
	taggs := parseTagg(data)
	if got := countNonZeroOffsets(taggs); got < 2 {
		t.Fatalf("expected mip chain with at least 2 levels, got %d", got)
	}

	off := firstMipOffset(taggs)
	if off == 0 || off+7 > len(data) {
		t.Fatalf("invalid first mip offset")
	}

	size := int(data[off+4]) | int(data[off+5])<<8 | int(data[off+6])<<16
	end := off + 7 + size
	if end > len(data) {
		t.Fatalf("first mip exceeds file bounds")
	}

	truncated := append([]byte(nil), data[:end]...)

	decoded, err := Decode(bytes.NewReader(truncated))
	if err != nil {
		t.Fatalf("Decode(truncated): %v", err)
	}
	if decoded.Bounds().Dx() != 64 || decoded.Bounds().Dy() != 64 {
		t.Fatalf("decoded size = %dx%d, want 64x64", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}

	cfg, err := DecodeConfig(bytes.NewReader(truncated))
	if err != nil {
		t.Fatalf("DecodeConfig(truncated): %v", err)
	}
	if cfg.Width != 64 || cfg.Height != 64 {
		t.Fatalf("config size = %dx%d, want 64x64", cfg.Width, cfg.Height)
	}
}

func TestToDDSPreservesMipmaps(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 8), G: uint8(y * 8), B: 128, A: uint8((x + y) & 0xFF)})
		}
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, &EncodeOptions{Type: PaxDXT5}); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}

	p, err := DecodePAA(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodePAA: %v", err)
	}

	dds, err := p.ToDDS()
	if err != nil {
		t.Fatalf("ToDDS: %v", err)
	}

	if dds.Format != bcn.FormatDXT5 {
		t.Fatalf("DDS format = %v, want %v", dds.Format, bcn.FormatDXT5)
	}
	if len(dds.Faces) != 1 {
		t.Fatalf("DDS faces = %d, want 1", len(dds.Faces))
	}
	if len(dds.Faces[0].Mipmaps) != len(p.MipMaps) {
		t.Fatalf("DDS mipmaps = %d, want %d", len(dds.Faces[0].Mipmaps), len(p.MipMaps))
	}

	for i := range p.MipMaps {
		if !bytes.Equal(dds.Faces[0].Mipmaps[i], p.MipMaps[i].Data) {
			t.Fatalf("mipmap[%d] payload differs", i)
		}
	}
}

func TestToKTXPreservesMipmaps(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 8), G: uint8(y * 8), B: 128, A: uint8((x + y) & 0xFF)})
		}
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, &EncodeOptions{Type: PaxDXT5}); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}

	p, err := DecodePAA(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodePAA: %v", err)
	}

	ktx, err := p.ToKTX()
	if err != nil {
		t.Fatalf("ToKTX: %v", err)
	}

	if ktx.Format != bcn.FormatDXT5 {
		t.Fatalf("KTX format = %v, want %v", ktx.Format, bcn.FormatDXT5)
	}
	if len(ktx.Faces) != 1 {
		t.Fatalf("KTX faces = %d, want 1", len(ktx.Faces))
	}
	if len(ktx.Faces[0].Mipmaps) != len(p.MipMaps) {
		t.Fatalf("KTX mipmaps = %d, want %d", len(ktx.Faces[0].Mipmaps), len(p.MipMaps))
	}

	for i := range p.MipMaps {
		if !bytes.Equal(ktx.Faces[0].Mipmaps[i], p.MipMaps[i].Data) {
			t.Fatalf("mipmap[%d] payload differs", i)
		}
	}

	var out bytes.Buffer
	if err := ktx.Write(&out); err != nil {
		t.Fatalf("KTX Write: %v", err)
	}

	readKTX, err := bcn.ReadKTX(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("ReadKTX: %v", err)
	}
	if len(readKTX.Faces) != 1 || len(readKTX.Faces[0].Mipmaps) != len(p.MipMaps) {
		t.Fatalf("read KTX shape mismatch")
	}

	for i := range p.MipMaps {
		if !bytes.Equal(readKTX.Faces[0].Mipmaps[i], p.MipMaps[i].Data) {
			t.Fatalf("read mipmap[%d] payload differs", i)
		}
	}
}

func TestDecodePAA_InvalidMagic(t *testing.T) {
	_, err := DecodePAA(bytes.NewReader([]byte{0, 0}))
	if err == nil {
		t.Fatal("DecodePAA with invalid magic should error")
	}
	if !errors.Is(err, ErrInvalidMagic) {
		t.Errorf("err = %v, want ErrInvalidMagic", err)
	}
}

func TestDecodePAA_NoSeeker(t *testing.T) {
	// Reader that doesn't implement Seeker: DecodePAA should ReadAll and use bytes.Reader.
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	data := buf.Bytes()

	type onlyReader struct{ io.Reader }
	r := onlyReader{Reader: bytes.NewReader(data)}

	decoded, err := Decode(r)
	if err != nil {
		t.Fatalf("Decode(non-Seeker): %v", err)
	}
	if decoded.Bounds().Dx() != 4 || decoded.Bounds().Dy() != 4 {
		t.Errorf("decoded size = %dx%d, want 4x4", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func TestNohqTagsBGRAAndCXAM(t *testing.T) {
	// Build a deterministic normal-like image.
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(10 + x), G: uint8(20 + y), B: 200, A: 255})
		}
	}

	opts := &EncodeOptions{
		Type:             PaxDXT5,
		NormalMapSwizzle: true,
		WriteSwizzleTag:  true,
		SwizzleTag:       [4]byte{0x05, 0x04, 0x02, 0x03},
	}
	noMips := false
	opts.GenerateMipmaps = &noMips
	opts.UseLZO = false

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, opts); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}
	data := buf.Bytes()
	taggs := parseTagg(data)

	cgva := taggs["CGVA"]
	cxam := taggs["CXAM"]
	ziws := taggs["ZIWS"]
	if len(cgva) != 4 || len(cxam) != 4 || len(ziws) != 4 {
		t.Fatalf("missing tags: CGVA=%d CXAM=%d ZIWS=%d", len(cgva), len(cxam), len(ziws))
	}

	// CGVA is stored in BGRA order; CXAM for _nohq is FF FF FF FF.
	if cgva[0] != 200 || cgva[1] != 23 || cgva[2] != 13 || cgva[3] != 255 {
		t.Fatalf("CGVA (BGRA) = %v, want [200 23 13 255]", cgva)
	}
	if cxam[0] != 0xFF || cxam[1] != 0xFF || cxam[2] != 0xFF || cxam[3] != 0xFF {
		t.Fatalf("CXAM = %v, want all 0xFF for _nohq", cxam)
	}
	if ziws[0] != 0x05 || ziws[1] != 0x04 || ziws[2] != 0x02 || ziws[3] != 0x03 {
		t.Fatalf("ZIWS = %v, want 05 04 02 03", ziws)
	}
}

func TestMipChainDefaults(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for i := range img.Pix {
		img.Pix[i] = 128
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, nil); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}

	p, err := DecodePAA(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodePAA: %v", err)
	}
	// 64,32,16,8,4 => 5 mips
	if len(p.MipMaps) != 5 {
		t.Fatalf("mip count=%d, want 5", len(p.MipMaps))
	}

	taggs := parseTagg(buf.Bytes())
	if got := countNonZeroOffsets(taggs); got != 5 {
		t.Fatalf("SFFO nonzero offsets=%d, want 5", got)
	}
}

func TestEncodeWithOptionsAndMetadataHeadersParity(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 7),
				G: uint8(y * 9),
				B: 160,
				A: 255,
			})
		}
	}

	opts := &EncodeOptions{
		Type:      PaxDXT5,
		WriteGALF: true,
		GALFValue: 3,
	}

	var buf bytes.Buffer
	metaFromEncode, err := EncodeWithOptionsAndMetadataHeaders(&buf, img, opts)
	if err != nil {
		t.Fatalf("EncodeWithOptionsAndMetadataHeaders: %v", err)
	}

	metaFromDecode, err := DecodeMetadataHeaders(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodeMetadataHeaders: %v", err)
	}

	if metaFromEncode.Type != metaFromDecode.Type {
		t.Fatalf("type mismatch: encode=%d decode=%d", metaFromEncode.Type, metaFromDecode.Type)
	}
	if metaFromEncode.HasAverageColor != metaFromDecode.HasAverageColor || metaFromEncode.AverageColor != metaFromDecode.AverageColor {
		t.Fatalf("CGVA mismatch: encode=%v decode=%v", metaFromEncode.AverageColor, metaFromDecode.AverageColor)
	}
	if metaFromEncode.HasMaxColor != metaFromDecode.HasMaxColor || metaFromEncode.MaxColor != metaFromDecode.MaxColor {
		t.Fatalf("CXAM mismatch: encode=%v decode=%v", metaFromEncode.MaxColor, metaFromDecode.MaxColor)
	}
	if metaFromEncode.HasGALF != metaFromDecode.HasGALF || metaFromEncode.GALF != metaFromDecode.GALF {
		t.Fatalf("GALF mismatch: encode=(%v,%d) decode=(%v,%d)", metaFromEncode.HasGALF, metaFromEncode.GALF, metaFromDecode.HasGALF, metaFromDecode.GALF)
	}
	if len(metaFromEncode.MipHeaders) != len(metaFromDecode.MipHeaders) {
		t.Fatalf("mipmap count mismatch: encode=%d decode=%d", len(metaFromEncode.MipHeaders), len(metaFromDecode.MipHeaders))
	}
	for i := range metaFromEncode.MipHeaders {
		if metaFromEncode.MipHeaders[i] != metaFromDecode.MipHeaders[i] {
			t.Fatalf("mipmap[%d] mismatch: encode=%+v decode=%+v", i, metaFromEncode.MipHeaders[i], metaFromDecode.MipHeaders[i])
		}
	}
}

func TestPerMipLZOFlag(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for i := range img.Pix {
		img.Pix[i] = 0
	}

	opts := &EncodeOptions{UseLZO: true}
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, opts); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}
	data := buf.Bytes()
	taggs := parseTagg(data)
	off := firstMipOffset(taggs)
	if off == 0 || off+4 >= len(data) {
		t.Fatalf("invalid SFFO offset")
	}
	rawW := uint16(data[off]) | uint16(data[off+1])<<8
	if (rawW & 0x8000) == 0 {
		t.Fatalf("expected LZO flag on first mip width, got %04x", rawW)
	}
}

func TestDecodeMasksLZOWidthBit(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	opts := &EncodeOptions{UseLZO: true, GenerateMipmaps: ptrBool(false)}
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, opts); err != nil {
		t.Fatalf("EncodeWithOptions: %v", err)
	}

	taggs := parseTagg(buf.Bytes())
	off := firstMipOffset(taggs)
	rawW := uint16(buf.Bytes()[off]) | uint16(buf.Bytes()[off+1])<<8
	if (rawW & 0x8000) == 0 {
		t.Fatalf("expected LZO flag on width, got %04x", rawW)
	}

	p, err := DecodePAA(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodePAA: %v", err)
	}
	if len(p.MipMaps) != 1 {
		t.Fatalf("mip count=%d, want 1", len(p.MipMaps))
	}
	if p.MipMaps[0].Width != 16 || p.MipMaps[0].Height != 16 {
		t.Fatalf("decoded size=%dx%d, want 16x16", p.MipMaps[0].Width, p.MipMaps[0].Height)
	}
}

func TestRoundTripHeadersFromPAAFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fixture round-trip headers test in short mode")
	}

	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "test_") && strings.HasSuffix(name, ".paa") {
			files = append(files, filepath.Join("testdata", name))
		}
	}
	if len(files) == 0 {
		t.Fatalf("no testdata/test_*.paa files found")
	}

	cfg, cfgErr := texconfig.DefaultTexConvertConfig()
	if cfgErr != nil {
		t.Fatalf("default texconfig: %v", cfgErr)
	}
	q := bcn.QualityLevelFast
	override := &EncodeOptions{BCn: &bcn.EncodeOptions{QualityLevel: q}}

	for _, path := range files {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer func() { _ = f.Close() }()

			img, err := Decode(f)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}

			name := filepath.Base(path)
			if _, ok := texconfig.Resolve(name, cfg); !ok {
				t.Skipf("no texconfig hint for %s (not in base config)", name)
			}
			var buf bytes.Buffer
			if err := EncodeWithTexConfigOptions(&buf, img, name, cfg, override); err != nil {
				t.Fatalf("encode: %v", err)
			}

			data := buf.Bytes()
			taggs := parseTagg(data)
			if len(taggs["CGVA"]) != 4 || len(taggs["CXAM"]) != 4 {
				t.Fatalf("missing CGVA/CXAM")
			}
			if len(taggs["SFFO"]) < 4 {
				t.Fatalf("missing SFFO")
			}
			assertSFFOOffsets(t, data, taggs)
		})
	}
}

func TestLZSSChecksumSigned(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = 128
	}

	opts := &EncodeOptions{
		Type:            PaxARGBA5,
		ForceLZSS:       true,
		GenerateMipmaps: ptrBool(false),
	}
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, opts); err != nil {
		t.Fatalf("encode: %v", err)
	}

	payload, _, _, _, err := firstMipPayload(buf.Bytes())
	if err != nil {
		t.Fatalf("firstMipPayload: %v", err)
	}
	if len(payload) < 4 {
		t.Fatalf("payload too short")
	}
	stored := binary.LittleEndian.Uint32(payload[len(payload)-4:])

	raw, err := encodePixelFormat(PaxARGBA5, img)
	if err != nil {
		t.Fatalf("encodePixelFormat: %v", err)
	}
	var sum int32
	for _, b := range raw {
		sum += int32(int8(b))
	}
	if stored != uint32(sum) {
		t.Fatalf("LZSS checksum mismatch: got=%08x want=%08x", stored, uint32(sum))
	}
}

func TestSFFOOffsetsMonotonic(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	taggs := parseTagg(buf.Bytes())
	assertSFFOOffsets(t, buf.Bytes(), taggs)
}

func TestNonDXTMipChain(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	opts := &EncodeOptions{Type: PaxARGB4}
	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, img, opts); err != nil {
		t.Fatalf("encode: %v", err)
	}
	p, err := DecodePAA(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodePAA: %v", err)
	}
	if len(p.MipMaps) != 4 { // 8,4,2,1
		t.Fatalf("mip count=%d, want 4", len(p.MipMaps))
	}
}


func assertSFFOOffsets(t *testing.T, data []byte, taggs map[string][]byte) {
	t.Helper()
	sffo := taggs["SFFO"]
	if len(sffo) < 4 {
		t.Fatalf("missing SFFO")
	}
	last := uint32(0)
	for i := 0; i+4 <= len(sffo); i += 4 {
		v := binary.LittleEndian.Uint32(sffo[i : i+4])
		if v == 0 {
			continue
		}
		if v <= last {
			t.Fatalf("SFFO offsets not increasing: %d then %d", last, v)
		}
		if int(v) >= len(data) {
			t.Fatalf("SFFO offset out of bounds: %d >= %d", v, len(data))
		}
		last = v
	}
}

func firstMipPayload(data []byte) (payload []byte, w, h int, lzo bool, err error) {
	taggs := parseTagg(data)
	off := firstMipOffset(taggs)
	if off == 0 || off+7 > len(data) {
		return nil, 0, 0, false, ErrInsufficientData
	}
	rawW := binary.LittleEndian.Uint16(data[off : off+2])
	rawH := binary.LittleEndian.Uint16(data[off+2 : off+4])
	lzo = (rawW & 0x8000) != 0
	w = int(rawW & 0x7FFF)
	h = int(rawH)
	size := int(data[off+4]) | int(data[off+5])<<8 | int(data[off+6])<<16
	if off+7+size > len(data) {
		return nil, 0, 0, false, ErrInsufficientData
	}
	return data[off+7 : off+7+size], w, h, lzo, nil
}

func ptrBool(v bool) *bool {
	return &v
}
