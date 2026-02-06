# paa

Go package for reading and writing **PAA** (Arma/DayZ texture) files.
Supports DXT1/DXT5 and several uncompressed formats, mipmaps with LZO/LZSS,
and optional registration with the standard `image` package.

## Features

* Decode and encode PAA (DXT1, DXT5, ARGB8, ARGB1555, ARGB4444, GRAYA/AI88)
* Mipmap support; LZO and LZSS decompression for mip data
* Optional registration with `image` via `paa/img`
* TexConvert.cfg‑style resolution via `texconfig` (suffix → format/swizzle/etc.)

## Usage

### With `image.Decode`

Import the **img** subpackage so that PAA is registered:

```go
import (
  _ "github.com/woozymasta/paa/img"
  "image"
)

img, format, err := image.Decode(f)
cfg, _, err := image.DecodeConfig(f)
```

### Direct API

```go
import "github.com/woozymasta/paa"

p, err := paa.DecodePAA(r)
img, err := p.MipMaps[0].Image()

err = paa.Encode(w, img)
```

### TexConvert.cfg‑style encoding

Use `texconfig` to resolve filename hints and apply swizzle/format rules:

```go
import (
  "github.com/woozymasta/paa"
  "github.com/woozymasta/paa/texconfig"
)

cfg, _ := texconfig.DefaultTexConvertConfig()
err := paa.EncodeWithTexConfig(w, img, "my_texture_nohq.paa", cfg)
```

You can also parse a real `TexConvert.cfg` and override values before encoding:

```go
cfg, _ := texconfig.ParseFile("TexConvert.cfg")
cfg.DisableAutoReduce = true
err := paa.EncodeWithTexConfig(w, img, "my_texture_nohq.paa", cfg)
```

### Encoding options

For explicit control, use `EncodeWithOptions`:

```go
opts := &paa.EncodeOptions{
  Type:          paa.PaxDXT5,
  UseLZO:        true,
  ForceCXAMFull: true,
  BCn: &bcn.EncodeOptions{
    QualityLevel: bcn.QualityLevelBest,
  },
}
err := paa.EncodeWithOptions(w, img, opts)
```
