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

### Metadata-only fast path

If you only need texture headers/tags (for example, building `texheaders`)
without decoding mip payloads, use metadata helpers:

```go
// full GGAT map + mip headers
meta, err := paa.DecodeMetadata(r)
// Type + mip headers + CGVA/CXAM/GALF only
headers, err := paa.DecodeMetadataHeaders(r)

meta2, err := paa.DecodeMetadataBytes(data)
headers2, err := paa.DecodeMetadataHeadersBytes(data)
```

For on-the-fly encode pipelines, you can collect compact metadata in the same
encode pass (without reading the generated PAA again):

```go
headers, err := paa.EncodeWithOptionsAndMetadataHeaders(w, img, opts)
```

### PAA -> DDS/KTX (preserve mipmaps)

If you need DDS/KTX with the original mip chain (without re-generating mipmaps),
use full PAA decode and convert:

```go
import (
  "github.com/woozymasta/paa"
)

p, err := paa.DecodePAA(r)
dds, err := p.ToDDS()
err = dds.Write(w)

ktx, err := p.ToKTX()
err = ktx.Write(w)
```

Or use a single-step helper:

```go
dds, err := paa.DecodeToDDS(r)
err = dds.Write(w)

ktx, err := paa.DecodeToKTX(r)
err = ktx.Write(w)
```

> [!NOTE]  
> direct conversion supports DXT-based PAA types.

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

## Concurrency and performance

Block (de)compression is parallelized across workers (default: `GOMAXPROCS`),
controlled by `Workers` in the BCn encode/decode options.
A few rules of thumb:

* **Single texture, lowest latency:** keep the default (auto).
  Encode scales well up to the number of physical cores;
  beyond that (SMT threads) the gains are small.
* **Decoding:** decode is dominated by LZO decompression and memory traffic,
  so block parallelism helps little and regresses past ~4 workers.
  The default is already capped at `min(GOMAXPROCS, 4)`;
  set `DecodeOptions.BCn.Workers` explicitly only to override it
  (e.g. `1` for batch).
* **Batch / many files:** prefer parallelism at the *file* level
  (one goroutine per file) with `Workers: 1` per call.
  Per-image scaling is sublinear, so spreading cores across files
  gives much higher total throughput than per-image parallelism.

### Reusable Encoder/Decoder

For batch pipelines, reuse a `paa.Encoder`/`paa.Decoder`
per worker goroutine to avoid re-allocating the mip chain,
BCn and (de)compression buffers on every image.
Buffers grow to fit the largest image seen and are reused for smaller ones,
cutting steady-state allocations by roughly 99% on large textures.

```go
enc := paa.NewEncoder() // one per goroutine; NOT safe for concurrent use
for _, job := range jobs {
    if err := enc.EncodeWithOptions(job.w, job.img, opts); err != nil {
        // handle err
    }
}

dec := paa.NewDecoder()
img, err := dec.Decode(r) // valid only until the next Decode on this dec
```

The package-level `Encode`/`Decode`
functions allocate per call and remain the simplest choice for one-off use.
