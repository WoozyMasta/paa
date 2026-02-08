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
