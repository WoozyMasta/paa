<!-- markdownlint-disable MD024 -->
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][],
and this project adheres to [Semantic Versioning][].

<!--
## Unreleased

### Added
### Changed
### Removed
-->

## Unreleased

### Added

* Zero-copy DDS/KTX export support for `PaxARGBA5` (`RGBA5551`)
  and `PaxARGB8` (`BGRA8`), preserving all mip payloads.

### Fixed

* `PaxDXT3` encoding to use BC2 payloads;
  `PaxDXT2` and `PaxDXT4` encoding now return `ErrUnsupportedFormat`
  until premultiplied-alpha conversion is implemented.

## [0.4.0][] - 2026-06-21

### Added

* `EncodeWithTexConfigFallback` like `EncodeWithTexConfig`
  but retries resolution with a `fallbackSuffix`-based name
  when the filename has no hint;
  returns `ErrUnsupportedFormat` if neither resolves.
* `EncodeWithFallback` same as above using the global default config.
* `texconfig.ResolveOrFallback` resolves a filename,
  falling back to `"texture_"+fallbackSuffix+".paa"` on no match.
* `EncodeFromDDS` and `EncodeFromKTX` for convert DDS/KTX with pre-compressed
  DXT1/DXT3/DXT5 blocks to PAA without re-encoding.
* `DecodeNthMip` decodes a single mip level by index (0-based)
  without loading others.
* `(*PAA).AllImages` decodes all mip levels from an already-parsed PAA;
  swizzle tags are applied consistently with `Decode`.
* `(*Decoder).DecodeMetadataHeaders` zero-alloc metadata scan in steady state;
  returns MetadataHeaders by value,
  reusing internal SFFO and mip-header buffers.
* `EncodeOptions.LZOLevel` controls LZO compression quality
  for DXT payloads when `UseLZO` is true:
  0/1 = fast LZO1X-1 (default), 2–9 = LZO1X-999 (better ratio, slower encode).
* `EncodeOptions.LZSSSearchLimit` overrides the LZSS backward-match window
  (default 2048, max 4096) for non-DXT payloads (AI88, ARGB4444, ARGB1555).

### Removed

* `EncodeOptions.ForceLZSS` - non-DXT mips are now always LZSS-compressed
  (the write-path fix made this flag a no-op).

### Fixed

* Non-square textures (e.g. 512x256) generated a spurious sub-minimum DXT mip:
  mip generation stopped only when both dimensions reached `MinMipSize` (4);
  it now stops when either does, matching the official ImageToPAA behaviour.
* Non-DXT mips (AI88, ARGB4444, ARGB1555) were stored uncompressed
  when LZSS would expand the data;
  TexView always expects LZSS regardless of size,
  causing "Out of memory" crashes on `_gs`, `_88`, `_1555`, and related types.
* `_8888`, `_raw`, and `_draftlco` texture types (all ARGB1555) were blocked by
  the encoder as a workaround for the LZSS crash above; they are now supported.

[0.4.0]: https://github.com/WoozyMasta/paa/compare/v0.3.0...v0.4.0

## [0.3.0][] - 2026-06-18

### Added

* `Encoder` and `Decoder` for allocation-free batch encoding/decoding:
  they reuse their buffers across calls (use one per goroutine).

### Changed

* Encode is about 2x faster and decode about 1.3x faster than v0.2.2,
  with significantly fewer allocations; output is unchanged.
  Reusing an `Encoder` or `Decoder` removes almost all
  remaining per-image allocations.
* Decode now uses at most 4 BCn worker threads by default
  (its LZO step is serial, so more workers only added overhead);
  override with `DecodeOptions.BCn.Workers`.
* Updated `bcn` to `v0.5.0`, `lzo` to `v0.3.0`, `lzss` to `v0.2.0`.

### Security

* Malformed PAA files can no longer trigger huge allocations:
  tag and mip sizes are validated against the file
  and the decoded size is capped `ErrTagSizeExceedsInput`, `ErrImageTooLarge`.

[0.3.0]: https://github.com/WoozyMasta/paa/compare/v0.2.2...v0.3.0

## [0.2.2][] - 2026-02-17

### Added

* `DecodeMetadataHeaders` / `DecodeMetadataHeadersBytes` for fast
  metadata-only reads (`Type`, `MipHeaders`, `CGVA`, `CXAM`, `GALF`)
  without keeping full GGAT map.
* `DecodeMetadataBytes` helper for zero-copy friendly in-memory metadata decode.
* `EncodeWithOptionsAndMetadataHeaders` for one-pass encode + compact metadata
  collection (no second decode pass of generated PAA).
* `MainFlowMetadataDecode` benchmark for metadata-only decode performance.

### Changed

* `texconfig` parse/validation sentinel errors are now centralized in
  `texconfig/errors.go` (instead of inline `errors.New(...)` at call sites).

[0.2.2]: https://github.com/WoozyMasta/paa/compare/v0.2.1...v0.2.2

## [0.2.1][] - 2026-02-17

### Changed

* Updated `github.com/woozymasta/lzss` to `v0.1.4`.
* Inherited LZSS decode/encode performance improvements from `lzss v0.1.4`,
  including reduced allocations in core paths only for Non-DXT PAA.

[0.2.1]: https://github.com/WoozyMasta/paa/compare/v0.2.0...v0.2.1

## [0.2.0][] - 2026-02-17

### Added

* `DecodeToDDS`/`DecodeToKTX` and `(*PAA).ToDDS`/`(*PAA).ToKTX` helpers for
  direct `PAA -> DDS/KTX` conversion with preserved mipmap chain
  (DXT-based PAA types).
* Main-flow I/O benchmarks (`BenchmarkMainFlowEncode`,
  `BenchmarkMainFlowDecode`, `BenchmarkMainFlowRoundTrip`).

### Changed

* Updated `github.com/woozymasta/lzo` to `v0.2.0`.
* `lzo v0.2.0` no longer includes legacy GPLv2 code
  and is distributed under MIT.
* Project licensing constraints are now MIT-only;
  previous GPLv2-related limitation is removed.
* LZO performance improved in `v0.2.0`: compression is up to 2x faster and
  decompression is about 4-8x faster (depending on usage pattern).
* DXT mip decode now uses `lzo.DecompressNInto` with preallocated output
  buffers and strict full-input-consumption validation.
* `Decode`/`DecodeWithOptions` now read and decode only the first mip level
  instead of decoding all mipmaps.
* `DecodeConfig` now reads only the first mip header instead of scanning
  all mip payload metadata.
* `EncodeWithOptions` image stats pass now has a zero-allocation fast path
  for `*image.NRGBA`, significantly reducing allocations in encode flow.

[0.2.0]: https://github.com/WoozyMasta/paa/compare/v0.1.2...v0.2.0

## [0.1.2][] - 2026-02-08

### Added

* `DecodeMetadata` function for scanning PAA file metadata without
  decoding full image data.

[0.1.2]: https://github.com/WoozyMasta/paa/compare/v0.1.0...v0.1.2

## [0.1.1][] - 2026-02-06

### Added

* `EncodeOptions.BCn` passthrough to `bcn.EncodeOptions` for full control.
* `DecodeOptions` and `DecodeWithOptions` to pass BCn decode settings.
* `MipMap.ImageWithOptions` for BCn decode options.
* `ErrorMetricsNormalMap` in `texconfig` for normal map tuning.

### Changed

* Normal map hints now use `ErrorMetricsNormalMap`, mapping to `RGBWeights 5/5/5`.
* README examples updated for new BCn options.
* BCn updated to more productive version 0.1.3 with support for parallelism.

### Removed (Breaking)

* `EncodeOptions.QualityLevel` and `EncodeOptions.RGBWeights`
  (use `EncodeOptions.BCn` instead).

[0.1.1]: https://github.com/WoozyMasta/paa/compare/v0.1.0...v0.1.1

## [0.1.0][] - 2026-02-04

### Added

* First public release

[0.1.0]: https://github.com/WoozyMasta/paa/tree/v0.1.0

<!--links-->
[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
[Semantic Versioning]: https://semver.org/spec/v2.0.0.html
