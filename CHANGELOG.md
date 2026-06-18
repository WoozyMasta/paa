<!-- markdownlint-disable MD024 -->
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][],
and this project adheres to [Semantic Versioning][].

## [0.3.0][] - 2026-06-18

### Added

* `Encoder` and `Decoder` for allocation-free batch encoding/decoding:
  they reuse their buffers across calls (use one per goroutine).

### Changed

* Encode is about 2x faster and decode about 1.3x faster than v0.2.2,
  with significantly fewer allocations; output is unchanged.
  Reusing an `Encoder` or `Decoder` removes almost all
  remaining per-image allocations.
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
