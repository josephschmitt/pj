# Changelog

## [1.1.0](https://github.com/josephschmitt/pj/compare/v1.0.2...v1.1.0) (2026-01-22)


### Features

* add icons for all default project markers ([7eb6eec](https://github.com/josephschmitt/pj/commit/7eb6eec2106fa2f605305c67b376397ea5ee4ac7))
* add support for .gitignore and .ignore files  ([#7](https://github.com/josephschmitt/pj/issues/7)) ([40ce418](https://github.com/josephschmitt/pj/commit/40ce4184c8087e38da69414a82197882ea7ef453))


### Bug Fixes

* add vendorHash to fix Nix builds ([dfd1cc7](https://github.com/josephschmitt/pj/commit/dfd1cc7593daecf8a4a2786a15d4442c15850426))
* use highest priority marker when multiple markers exist ([cfbf76d](https://github.com/josephschmitt/pj/commit/cfbf76d0102043bd57b6dd2a63b48b087fce342c))

## [1.0.2](https://github.com/josephschmitt/pj/compare/v1.0.1...v1.0.2) (2026-01-22)


### Bug Fixes

* update homebrew tap repository name to homebrew-tap ([1705f62](https://github.com/josephschmitt/pj/commit/1705f623d922f679426376171c80693d44bf2891))

## [1.0.1](https://github.com/josephschmitt/pj/compare/v1.0.0...v1.0.1) (2026-01-22)


### Bug Fixes

* use dedicated PAT for release-please to trigger release workflow ([62bd3d4](https://github.com/josephschmitt/pj/commit/62bd3d48a4aa7ec23041ed6ecaf338145d1add45))

## 1.0.0 (2026-01-22)


### Features

* add automated releases with conventional commits ([4f942aa](https://github.com/josephschmitt/pj/commit/4f942aa9fe6b3c62adbedca7c1d89250e25a5a5b))
* Add comprehensive distribution infrastructure ([79746a3](https://github.com/josephschmitt/pj/commit/79746a351b043fee264b0b4d9035625bb9251480))
* Initial implementation of pj - Project Finder CLI ([c98ff5e](https://github.com/josephschmitt/pj/commit/c98ff5e0602e27930a644fdd38316b9c3d1e676e))


### Bug Fixes

* remove deprecated package-name from release-please config ([d859c31](https://github.com/josephschmitt/pj/commit/d859c31b083ef0f3e5b3eb14ce17f0d7871f0bde))
* remove path prefix from coverage.out for Windows compatibility ([2a39f3d](https://github.com/josephschmitt/pj/commit/2a39f3ddad54a43c9579e166dc9d9ec00d9dddb3))
* update CI coverage path and GoReleaser brew config ([27e38cf](https://github.com/josephschmitt/pj/commit/27e38cfe9d947878675b840e582affc57dbd675c))
* update GoReleaser config to remove deprecated properties ([6636700](https://github.com/josephschmitt/pj/commit/6636700a06a04f9774839082eadc0e171bfd956a))
* use bash shell for tests on all platforms ([1e58846](https://github.com/josephschmitt/pj/commit/1e58846d2813edac4a3a4dd7bb4ecaf26d05c236))
