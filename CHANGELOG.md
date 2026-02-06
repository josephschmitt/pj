# Changelog

## [1.7.1](https://github.com/josephschmitt/pj/compare/v1.7.0...v1.7.1) (2026-02-06)


### Bug Fixes

* **ci:** use jq instead of grep -P to read Go version from devbox.json ([0d862ff](https://github.com/josephschmitt/pj/commit/0d862ff511393a678621aeea0a04b500a6590c17))
* enable CGO for race detector in test-coverage target ([38157e5](https://github.com/josephschmitt/pj/commit/38157e54d5b3659559e67170b6aaea15823bced4))
* resolve errcheck lint errors in cache hash computation ([40dd4ad](https://github.com/josephschmitt/pj/commit/40dd4ad1f9c58ac8907df3ac29431a11f1ffccd7))
* resolve lint errors for errcheck and staticcheck ([ff29c06](https://github.com/josephschmitt/pj/commit/ff29c06aa1551d6f897f3d496471fd346c2345e9))

## [1.7.0](https://github.com/josephschmitt/pj/compare/v1.6.0...v1.7.0) (2026-02-04)


### Features

* add glob pattern support for markers ([#23](https://github.com/josephschmitt/pj/issues/23)) ([f1e292f](https://github.com/josephschmitt/pj/commit/f1e292f74b85033e4682630d88bc42637dc5aac2))

## [1.6.0](https://github.com/josephschmitt/pj/compare/v1.5.0...v1.6.0) (2026-02-02)


### Features

* add Dockerfile marker, configurable priority, and dev tooling ([#20](https://github.com/josephschmitt/pj/issues/20)) ([12c8858](https://github.com/josephschmitt/pj/commit/12c8858d3b9c1060881f46fab6d78610ed91a655))

## [1.5.0](https://github.com/josephschmitt/pj/compare/v1.4.1...v1.5.0) (2026-02-02)


### Features

* add new markers config format with inline icons ([#18](https://github.com/josephschmitt/pj/issues/18)) ([0fa8678](https://github.com/josephschmitt/pj/commit/0fa86786776cf82a3aac5621e85236e213d05988))
* notify pj-node on major/minor releases for npm sync ([c83002b](https://github.com/josephschmitt/pj/commit/c83002b0b98801754e14dfd16df293b69bf9d105))

## [1.4.1](https://github.com/josephschmitt/pj/compare/v1.4.0...v1.4.1) (2026-01-31)


### Bug Fixes

* update README download URLs to v1.4.0 and automate future updates ([#15](https://github.com/josephschmitt/pj/issues/15)) ([bf723e0](https://github.com/josephschmitt/pj/commit/bf723e0eaa4c1d8359ea8f6cb6fd7f486f267017))

## [1.4.0](https://github.com/josephschmitt/pj/compare/v1.3.0...v1.4.0) (2026-01-24)


### Features

* add JSON output mode with marker field for editor integration  ([#12](https://github.com/josephschmitt/pj/issues/12)) ([9de663c](https://github.com/josephschmitt/pj/commit/9de663cb19c903d307fba8349ee14ee10a5fe744))

## [1.3.0](https://github.com/josephschmitt/pj/compare/v1.2.0...v1.3.0) (2026-01-22)


### Features

* add IDE project markers for VS Code, IntelliJ, Fleet, Eclipse, and Zed ([#9](https://github.com/josephschmitt/pj/issues/9)) ([583f4bb](https://github.com/josephschmitt/pj/commit/583f4bbe005b2d5a75da3f4e6381733a9a2a2905))
* add nested project discovery for monorepo support ([#11](https://github.com/josephschmitt/pj/issues/11)) ([b6ff94a](https://github.com/josephschmitt/pj/commit/b6ff94a9076372cddf20b92b0e30b1b3e7ba77a6))


### Bug Fixes

* switch release-please to manifest mode to prevent duplicate release errors ([073244a](https://github.com/josephschmitt/pj/commit/073244a8ea0ad07c641ffe4dbe8f2744f8f48e77))

## [1.2.0](https://github.com/josephschmitt/pj/compare/v1.1.0...v1.2.0) (2026-01-22)


### Features

* add stdin support for Unix pipeline integration ([8a5434d](https://github.com/josephschmitt/pj/commit/8a5434d0626e018835a2cd3095069b1a41e4a0f5))

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
