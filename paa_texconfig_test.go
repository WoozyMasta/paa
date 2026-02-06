package paa

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/woozymasta/paa/texconfig"
)

func TestTexConfigHeadersFromPNG(t *testing.T) {
	cfg, cfgErr := texconfig.DefaultTexConvertConfig()
	if cfgErr != nil {
		t.Fatalf("default texconfig: %v", cfgErr)
	}

	tests := makeTestCases()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			img := tc.gen()
			hint, ok := texconfig.Resolve(tc.name, cfg)
			if !ok {
				t.Fatalf("no texconfig hint for %s", tc.name)
			}
			if isTexViewUnsupported(hint) {
				t.Skipf("unsupported in TexView: %s", tc.name)
			}

			var buf bytes.Buffer
			if err := EncodeWithTexConfigOptions(&buf, img, tc.name, cfg, nil); err != nil {
				t.Fatalf("encode: %v", err)
			}

			data := buf.Bytes()
			if len(data) < 2 {
				t.Fatalf("encoded data too short")
			}
			pax, ok := PaxTypeFromBytes(data[:2])
			if !ok {
				t.Fatalf("invalid PaxType")
			}

			taggs := parseTagg(data)
			if len(taggs["CGVA"]) != 4 {
				t.Fatalf("missing CGVA")
			}
			if len(taggs["CXAM"]) != 4 {
				t.Fatalf("missing CXAM")
			}
			if len(taggs["SFFO"]) < 4 {
				t.Fatalf("missing SFFO")
			}

			if isDXT(pax) || pax == PaxARGB4 || pax == PaxARGBA5 {
				cxam := taggs["CXAM"]
				if cxam[0] != 0xFF || cxam[1] != 0xFF || cxam[2] != 0xFF || cxam[3] != 0xFF {
					t.Fatalf("CXAM not forced to FF for %v: %v", pax, cxam)
				}
			}

			if !hint.Swizzle.IsIdentity() {
				vs := hint.VirtualSwz == nil || *hint.VirtualSwz
				if vs {
					if len(taggs["ZIWS"]) != 4 {
						t.Fatalf("missing ZIWS for swizzled hint")
					}
				}
			}

			if hint.ClassName == "detail" || hint.ClassName == "detail_short" {
				galf := taggs["GALF"]
				if len(galf) != 4 || galf[0] != 2 {
					t.Fatalf("detail map should have GALF=2, got %v", galf)
				}
				ziws := taggs["ZIWS"]
				if len(ziws) != 4 || ziws[0] != 0x08 || ziws[1] != 0x00 || ziws[2] != 0x00 || ziws[3] != 0x00 {
					t.Fatalf("detail map ZIWS mismatch: %v", ziws)
				}
			}

			off := firstMipOffset(taggs)
			if off == 0 || off+7 >= len(data) {
				t.Fatalf("invalid SFFO offset %d", off)
			}
			w := int(binary.LittleEndian.Uint16(data[off:off+2]) & 0x7FFF)
			h := int(binary.LittleEndian.Uint16(data[off+2 : off+4]))
			size := int(data[off+4]) | int(data[off+5])<<8 | int(data[off+6])<<16
			if w <= 0 || h <= 0 || size <= 0 {
				t.Fatalf("invalid mip header w=%d h=%d size=%d", w, h, size)
			}
			if off+7+size > len(data) {
				t.Fatalf("mip payload out of bounds")
			}
		})
	}
}

type testCase struct {
	name string
	gen  func() image.Image
}

func makeTestCases() []testCase {
	return []testCase{
		{name: "test_co.paa", gen: genColor},
		{name: "test_ca.paa", gen: genColorAlpha},
		// {name: "test_alpha.paa", gen: genAlphaOnly}, // Not in TexConvert.cfg, still supported.
		{name: "test_lco.paa", gen: genColor},
		{name: "test_lca.paa", gen: genMaskPalette},
		{name: "test_mask.paa", gen: genMaskPalette},
		{name: "test_mc.paa", gen: genMacro},
		{name: "test_mco.paa", gen: genMCONoise},
		{name: "test_cdt.paa", gen: genColoredDetail},
		{name: "test_dt.paa", gen: genDetail},
		{name: "test_detail.paa", gen: genDetail},
		{name: "test_dtsmdi.paa", gen: genDTSMDI},
		{name: "test_sm.paa", gen: genSpecular},
		{name: "test_smdi.paa", gen: genSpecular},
		{name: "test_can.paa", gen: genColor},
		{name: "test_cat.paa", gen: genColorAlpha},
		{name: "test_gs.paa", gen: genAlphaOnly},
		{name: "test_as.paa", gen: genAS},
		{name: "test_ads.paa", gen: genADS},
		// {name: "test_asd.paa", gen: genADS}, // Not in TexConvert.cfg, still supported.
		{name: "test_adshq.paa", gen: genADSHQ},
		{name: "test_pr.paa", gen: genPR},
		{name: "test_sky.paa", gen: genSky},
		// {name: "test_ti.paa", gen: genThermal}, // Not in TexConvert.cfg, still supported.
		{name: "test_no.paa", gen: genNormal},
		{name: "test_normalmap.paa", gen: genNormal},
		{name: "test_noex.paa", gen: genNormalSpec},
		{name: "test_nsex.paa", gen: genNormalSpec},
		{name: "test_nohq.paa", gen: genNormal},
		{name: "test_nshq.paa", gen: genNormalSpec},
		// {name: "test_nohq_alpha.paa", gen: genNormalAlpha}, // Not in TexConvert.cfg, still supported.
		{name: "test_nopx.paa", gen: genNormalParallax},
		{name: "test_nof.paa", gen: genNormal},
		{name: "test_nofhq.paa", gen: genNormal},
		{name: "test_nofex.paa", gen: genNormalSpec},
		{name: "test_non.paa", gen: genNormalNoise},
		{name: "test_novhq.paa", gen: genNormal},
		{name: "test_ns.paa", gen: genNormalSpec},
		{name: "test_dxt1.paa", gen: genColor},
		{name: "test_dxt5.paa", gen: genColorAlpha},
		{name: "test_4444.paa", gen: genColorAlpha},
		{name: "test_1555.paa", gen: genMaskPalette},
		{name: "test_88.paa", gen: genAlphaOnly},
		{name: "test_8888.paa", gen: genColorAlpha},
		{name: "test_raw.paa", gen: genColor},
		{name: "test_draftlco.paa", gen: genColor},
	}
}

const testImgSize = 64

func genColor() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			r := uint8((x * 255) / (testImgSize - 1))
			g := uint8((y * 255) / (testImgSize - 1))
			b := uint8(((x + y) / 2 * 255) / (testImgSize - 1))
			if (x/8+y/8)%2 == 0 {
				b = 255 - b
			}
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func genColorAlpha() image.Image {
	img := genColor().(*image.NRGBA)
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			alpha := uint8((x + y) * 255 / (2 * (testImgSize - 1)))
			c := img.NRGBAAt(x, y)
			c.A = alpha
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genAlphaOnly() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			alpha := uint8((x * 255) / (testImgSize - 1))
			img.SetNRGBA(x, y, color.NRGBA{R: 128, G: 128, B: 128, A: alpha})
		}
	}
	return img
}

func genMaskPalette() image.Image {
	palette := []color.NRGBA{
		{R: 255, G: 0, B: 0, A: 255},
		{R: 0, G: 255, B: 0, A: 255},
		{R: 0, G: 0, B: 255, A: 255},
		{R: 255, G: 255, B: 0, A: 255},
		{R: 255, G: 0, B: 255, A: 255},
		{R: 0, G: 255, B: 255, A: 255},
	}
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	cell := testImgSize / 4
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			pi := ((y / cell) * 4) + (x / cell)
			pi = pi % len(palette)
			img.SetNRGBA(x, y, palette[pi])
		}
	}
	return img
}

func genMCONoise() image.Image {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			v := uint8(rng.Intn(256))
			img.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

func genMacro() image.Image {
	img := genColor().(*image.NRGBA)
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			c := img.NRGBAAt(x, y)
			alpha := uint8((x * 255) / (testImgSize - 1))
			c.A = alpha
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genDetail() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			alpha := uint8((x ^ y) & 0xFF)
			img.SetNRGBA(x, y, color.NRGBA{R: 128, G: 128, B: 128, A: alpha})
		}
	}
	return img
}

func genColoredDetail() image.Image {
	img := genColor().(*image.NRGBA)
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			c := img.NRGBAAt(x, y)
			c.R = uint8((int(c.R) + 128) / 2)
			c.G = uint8((int(c.G) + 128) / 2)
			c.B = uint8((int(c.B) + 128) / 2)
			c.A = 255
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genDTSMDI() image.Image {
	img := genSpecular().(*image.NRGBA)
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			c := img.NRGBAAt(x, y)
			c.R = uint8((x * 255) / (testImgSize - 1))
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genSpecular() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			r := uint8((x * 255) / (testImgSize - 1))
			g := uint8((y * 255) / (testImgSize - 1))
			b := uint8(((x + y) * 255) / (2 * (testImgSize - 1)))
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func genAS() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			g := uint8((x * 255) / (testImgSize - 1))
			img.SetNRGBA(x, y, color.NRGBA{R: 0, G: g, B: 0, A: 255})
		}
	}
	return img
}

func genADS() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			g := uint8((x * 255) / (testImgSize - 1))
			b := uint8((y * 255) / (testImgSize - 1))
			img.SetNRGBA(x, y, color.NRGBA{R: 0, G: g, B: b, A: 255})
		}
	}
	return img
}

func genADSHQ() image.Image {
	return genADS()
}

func genPR() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			r := uint8((x * 255) / (testImgSize - 1))
			g := uint8(((x + y) * 255) / (2 * (testImgSize - 1)))
			b := uint8((y * 255) / (testImgSize - 1))
			a := uint8((255 - r + b) / 2)
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img
}

func genSky() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			t := float64(y) / float64(testImgSize-1)
			r := uint8(40 + t*40)
			g := uint8(80 + t*60)
			b := uint8(160 + t*80)
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func genThermal() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			r := uint8((x * 255) / (testImgSize - 1))
			g := uint8((y * 255) / (testImgSize - 1))
			b := uint8(((x + y) * 255) / (2 * (testImgSize - 1)))
			a := uint8((255 - r + g) / 2)
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img
}

func genNormal() image.Image {
	return genNormalBase(false, false)
}

func genNormalAlpha() image.Image {
	return genNormalBase(true, false)
}

func genNormalParallax() image.Image {
	return genNormalBase(true, false)
}

func genNormalNoise() image.Image {
	return genNormalBase(false, true)
}

func genNormalSpec() image.Image {
	img := genNormal().(*image.NRGBA)
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			c := img.NRGBAAt(x, y)
			c.A = uint8((x * 255) / (testImgSize - 1))
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genNormalBase(alphaGradient, addNoise bool) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, testImgSize, testImgSize))
	rng := rand.New(rand.NewSource(1))
	for y := 0; y < testImgSize; y++ {
		for x := 0; x < testImgSize; x++ {
			u := (float64(x)/float64(testImgSize-1))*2.0 - 1.0
			v := (float64(y)/float64(testImgSize-1))*2.0 - 1.0

			var nx, ny, nz float64
			if x < testImgSize/2 && y < testImgSize/2 {
				lu := u*2.0 + 1.0
				lv := v*2.0 + 1.0
				d := 1.0 - lu*lu - lv*lv
				if d > 0 {
					nx, ny, nz = lu, lv, math.Sqrt(d)
				} else {
					nx, ny, nz = 0, 0, 1
				}
			} else if x >= testImgSize/2 && y < testImgSize/2 {
				nx, ny, nz = 0, 0, 1
			} else if x < testImgSize/2 && y >= testImgSize/2 {
				angle := (float64(x) / float64(testImgSize/2)) * (math.Pi / 2)
				nx = math.Sin(angle)
				ny = 0
				nz = math.Cos(angle)
			} else {
				angle := ((float64(y) - float64(testImgSize/2)) / float64(testImgSize/2)) * (math.Pi / 2)
				nx = 0
				ny = math.Sin(angle)
				nz = math.Cos(angle)
			}

			if addNoise {
				nx += (rng.Float64() - 0.5) * 0.08
				ny += (rng.Float64() - 0.5) * 0.08
				nz = math.Sqrt(math.Max(0.0, 1.0-nx*nx-ny*ny))
			}

			r := uint8((nx*0.5 + 0.5) * 255)
			g := uint8((ny*0.5 + 0.5) * 255)
			b := uint8((nz*0.5 + 0.5) * 255)

			a := uint8(255)
			if alphaGradient {
				if x >= testImgSize/2 && y < testImgSize/2 {
					fx := float64(x-testImgSize/2) / float64(testImgSize/2-1)
					fy := float64((testImgSize/2-1)-y) / float64(testImgSize/2-1)
					a = uint8(clamp(int((fx+fy)*0.5*255), 0, 255))
				}
			}

			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
