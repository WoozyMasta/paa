# paa

[paa] is a Go library for reading and writing [PAA][paa-format] textures
used by Arma and DayZ.

It helps asset tools and conversion pipelines work
with game textures without losing their mipmaps, metadata,
or format-specific encoding choices.

Use `paa` to inspect and convert PAA files, create textures from Go images,
or move compressed textures between PAA, DDS, and KTX.
The package also supports TexConvert.cfg-style filename rules
for choosing formats and swizzles during encoding.

## Features

* Decode and encode PAA (DXT1, DXT5, ARGB8, ARGB1555, ARGB4444, GRAYA/AI88)
* Mipmap support; LZO and LZSS decompression for mip data
* Optional registration with `image` via `paa/img`
* TexConvert.cfg‑style resolution via `texconfig` (suffix → format/swizzle/etc.)

## Installation

Add the dependency with:

```sh
go get github.com/woozymasta/paa@latest
```

## Use

Decode a PAA texture and encode a Go image:

```go
func convert(r io.Reader, w io.Writer) error {
    texture, err := paa.DecodePAA(r)
    if err != nil {
        return err
    }

    img, err := texture.MipMaps[0].Image()
    if err != nil {
        return err
    }

    return paa.Encode(w, img)
}
```

For standard-library image decoding,
import `github.com/woozymasta/paa/img` to register PAA with `image.Decode`.
Batch tools can reuse an `Encoder` or `Decoder` per worker
to reuse their working buffers.

## Capabilities

* Decode and encode DXT1, DXT3, DXT5,
  and uncompressed ARGB8, ARGB1555, ARGB4444, and GRAYA/AI88 formats.
* Read mipmaps, texture tags, and headers;
  metadata helpers can skip mip payloads when pixel data is not needed.
* Convert DXT-based textures between PAA, DDS, and KTX
  while retaining the original mip chain.
* Apply TexConvert.cfg-style format, swizzle, and filename rules when encoding.
* Use reusable encoders and decoders in batch pipelines
  to reduce steady-state allocations.

The package-level API is intended for one-off operations.
`Encoder` and `Decoder` provide buffer reuse for repeated work.

## Concurrency and performance

Default worker settings suit a single texture.
For batches, parallelize across files and set BCn workers to `1` per call,
using one `Encoder` or `Decoder` per goroutine.
PAA decoding defaults to `min(GOMAXPROCS, 4)` BCn workers
because its LZO decompression step is serial.
Set a non-zero worker count in `DecodeOptions.BCn` to override it.

[paa]: https://pkg.go.dev/github.com/woozymasta/paa
[paa-format]: https://community.bistudio.com/wiki/PAA_File_Format
