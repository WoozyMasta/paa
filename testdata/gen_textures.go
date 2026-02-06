// Package main generates PNG test textures used by external tools.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

const imgSize = 128

type textureSpec struct {
	gen  func() *image.NRGBA
	name string
}

func main() {
	outDir := "testdata"
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		panic(err)
	}

	specs := []textureSpec{
		{name: "test_co.png", gen: genColor},
		{name: "test_ca.png", gen: genColorAlpha},
		{name: "test_alpha.png", gen: genAlphaOnly},
		{name: "test_lco.png", gen: genColor},
		// {name: "test_draftlco.png", gen: genColor}, // TODO: unsupported (TexView crash)
		{name: "test_lca.png", gen: genMaskPalette},
		{name: "test_mask.png", gen: genMaskPalette},
		{name: "test_mc.png", gen: genMacro},
		{name: "test_mco.png", gen: genMCONoise},
		{name: "test_cdt.png", gen: genColoredDetail},
		{name: "test_dt.png", gen: genDetail},
		{name: "test_detail.png", gen: genDetail},
		{name: "test_dtsmdi.png", gen: genDTSMDI},
		{name: "test_sm.png", gen: genSpecular},
		{name: "test_smdi.png", gen: genSpecular},
		{name: "test_can.png", gen: genColor},
		{name: "test_cat.png", gen: genColorAlpha},
		{name: "test_gs.png", gen: genAlphaOnly},
		{name: "test_as.png", gen: genAS},
		{name: "test_ads.png", gen: genADS},
		{name: "test_asd.png", gen: genADS},
		{name: "test_adshq.png", gen: genADSHQ},
		{name: "test_pr.png", gen: genPR},
		{name: "test_sky.png", gen: genSky},
		{name: "test_ti.png", gen: genThermal},
		{name: "test_no.png", gen: genNormal},
		{name: "test_normalmap.png", gen: genNormal},
		{name: "test_noex.png", gen: genNormalSpec},
		{name: "test_nsex.png", gen: genNormalSpec},
		{name: "test_nohq.png", gen: genNormal},
		{name: "test_nshq.png", gen: genNormalSpec},
		{name: "test_nohq_alpha.png", gen: genNormalAlpha},
		{name: "test_nopx.png", gen: genNormalParallax},
		{name: "test_nof.png", gen: genNormal},
		{name: "test_nofhq.png", gen: genNormal},
		{name: "test_nofex.png", gen: genNormalSpec},
		{name: "test_non.png", gen: genNormalNoise},
		{name: "test_novhq.png", gen: genNormal},
		{name: "test_ns.png", gen: genNormalSpec},
		{name: "test_dxt1.png", gen: genColor},
		{name: "test_dxt5.png", gen: genColorAlpha},
		{name: "test_4444.png", gen: genColorAlpha},
		{name: "test_1555.png", gen: genMaskPalette},
		{name: "test_88.png", gen: genAlphaOnly},
		// {name: "test_8888.png", gen: genColorAlpha}, // TODO: unsupported (TexView crash)
		// {name: "test_raw.png", gen: genColor},       // TODO: unsupported (TexView crash)
	}

	for _, spec := range specs {
		img := spec.gen()
		if err := writePNG(filepath.Join(outDir, spec.name), img); err != nil {
			panic(err)
		}
	}
	fmt.Printf("Wrote %d textures to %s\n", len(specs), outDir)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return png.Encode(f, img)
}

func genColor() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			r := u8((x * 255) / (imgSize - 1))
			g := u8((y * 255) / (imgSize - 1))
			b := u8(((x + y) / 2 * 255) / (imgSize - 1))
			if (x/16+y/16)%2 == 0 {
				b = 255 - b
			}
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func genColorAlpha() *image.NRGBA {
	img := genColor()
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			alpha := u8((x + y) * 255 / (2 * (imgSize - 1)))
			c := img.NRGBAAt(x, y)
			c.A = alpha
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genAlphaOnly() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			alpha := u8((x * 255) / (imgSize - 1))
			img.SetNRGBA(x, y, color.NRGBA{R: 128, G: 128, B: 128, A: alpha})
		}
	}
	return img
}

func genMaskPalette() *image.NRGBA {
	palette := []color.NRGBA{
		{R: 255, G: 0, B: 0, A: 255},
		{R: 0, G: 255, B: 0, A: 255},
		{R: 0, G: 0, B: 255, A: 255},
		{R: 255, G: 255, B: 0, A: 255},
		{R: 255, G: 0, B: 255, A: 255},
		{R: 0, G: 255, B: 255, A: 255},
	}
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	cell := imgSize / 4
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			pi := ((y / cell) * 4) + (x / cell)
			pi = pi % len(palette)
			img.SetNRGBA(x, y, palette[pi]) //nolint:gosec // G602: bounded by modulus
		}
	}
	return img
}

func genMCONoise() *image.NRGBA {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // G404: test data only
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			v := u8(rng.Intn(256))
			img.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

func genMacro() *image.NRGBA {
	img := genColor()
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			c := img.NRGBAAt(x, y)
			alpha := u8((x * 255) / (imgSize - 1))
			c.A = alpha
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genDetail() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			gray := uint8(128)
			alpha := u8((x ^ y) & 0xFF)
			img.SetNRGBA(x, y, color.NRGBA{R: gray, G: gray, B: gray, A: alpha})
		}
	}
	return img
}

func genColoredDetail() *image.NRGBA {
	img := genColor()
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			c := img.NRGBAAt(x, y)
			c.R = u8((int(c.R) + 128) / 2)
			c.G = u8((int(c.G) + 128) / 2)
			c.B = u8((int(c.B) + 128) / 2)
			c.A = 255
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genDTSMDI() *image.NRGBA {
	img := genSpecular()
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			c := img.NRGBAAt(x, y)
			c.R = u8((x * 255) / (imgSize - 1)) // detail in R
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genSpecular() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			r := u8((x * 255) / (imgSize - 1))
			g := u8((y * 255) / (imgSize - 1))
			b := u8(((x + y) * 255) / (2 * (imgSize - 1)))
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func genAS() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			g := u8((x * 255) / (imgSize - 1))
			img.SetNRGBA(x, y, color.NRGBA{R: 0, G: g, B: 0, A: 255})
		}
	}
	return img
}

func genADS() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			g := u8((x * 255) / (imgSize - 1))
			b := u8((y * 255) / (imgSize - 1))
			img.SetNRGBA(x, y, color.NRGBA{R: 0, G: g, B: b, A: 255})
		}
	}
	return img
}

func genADSHQ() *image.NRGBA {
	img := genADS()
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			c := img.NRGBAAt(x, y)
			c.A = 255
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genPR() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			r := u8((x * 255) / (imgSize - 1))
			g := u8(((x + y) * 255) / (2 * (imgSize - 1)))
			b := u8((y * 255) / (imgSize - 1))
			a := u8((int(255-r) + int(b)) / 2)
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img
}

func genSky() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			t := float64(y) / float64(imgSize-1)
			r := u8(int(40 + t*40))
			g := u8(int(80 + t*60))
			b := u8(int(160 + t*80))
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func genThermal() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			r := u8((x * 255) / (imgSize - 1))
			g := u8((y * 255) / (imgSize - 1))
			b := u8(((x + y) * 255) / (2 * (imgSize - 1)))
			a := u8((int(255-r) + int(g)) / 2)
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img
}

func genNormal() *image.NRGBA {
	return genNormalBase(false, false)
}

func genNormalAlpha() *image.NRGBA {
	return genNormalBase(true, false)
}

func genNormalParallax() *image.NRGBA {
	return genNormalBase(true, false)
}

func genNormalNoise() *image.NRGBA {
	return genNormalBase(false, true)
}

func genNormalSpec() *image.NRGBA {
	img := genNormal()
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			c := img.NRGBAAt(x, y)
			c.A = u8((x * 255) / (imgSize - 1))
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func genNormalBase(alphaGradient, addNoise bool) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, imgSize, imgSize))
	rng := rand.New(rand.NewSource(1)) //nolint:gosec // G404: deterministic noise for tests
	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			u := (float64(x)/float64(imgSize-1))*2.0 - 1.0
			v := (float64(y)/float64(imgSize-1))*2.0 - 1.0

			var nx, ny, nz float64
			if x < imgSize/2 && y < imgSize/2 {
				lu := u*2.0 + 1.0
				lv := v*2.0 + 1.0
				d := 1.0 - lu*lu - lv*lv
				if d > 0 {
					nx, ny, nz = lu, lv, math.Sqrt(d)
				} else {
					nx, ny, nz = 0, 0, 1
				}
			} else if x >= imgSize/2 && y < imgSize/2 {
				nx, ny, nz = 0, 0, 1
			} else if x < imgSize/2 && y >= imgSize/2 {
				angle := (float64(x) / float64(imgSize/2)) * (math.Pi / 2)
				nx = math.Sin(angle)
				ny = 0
				nz = math.Cos(angle)
			} else {
				angle := ((float64(y) - float64(imgSize/2)) / float64(imgSize/2)) * (math.Pi / 2)
				nx = 0
				ny = math.Sin(angle)
				nz = math.Cos(angle)
			}

			if addNoise {
				nx += (rng.Float64() - 0.5) * 0.08
				ny += (rng.Float64() - 0.5) * 0.08
				nz = math.Sqrt(math.Max(0.0, 1.0-nx*nx-ny*ny))
			}

			r := u8(int((nx*0.5 + 0.5) * 255))
			g := u8(int((ny*0.5 + 0.5) * 255))
			b := u8(int((nz*0.5 + 0.5) * 255))

			a := uint8(255)
			if alphaGradient {
				if x >= imgSize/2 && y < imgSize/2 {
					fx := float64(x-imgSize/2) / float64(imgSize/2-1)
					fy := float64((imgSize/2-1)-y) / float64(imgSize/2-1)
					a = u8(clamp(int((fx+fy)*0.5*255), 0, 255))
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

func u8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v) //nolint:gosec // G115: clamped to [0..255]
}
