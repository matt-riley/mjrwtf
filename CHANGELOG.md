# Changelog

## [0.10.0](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.9.0...mjrwtf-v0.10.0) (2026-01-13)


### Features

* **config:** add Tailscale configuration fields ([992e242](https://github.com/matt-riley/mjrwtf/commit/992e2426dd51bc9db743e73cdb4dcb95be7e5833))
* **config:** add Tailscale configuration fields ([9a03f3d](https://github.com/matt-riley/mjrwtf/commit/9a03f3d76205ecc9e451e3cf0aecd30a096d29d2))
* **deps:** add Tailscale tsnet and client libraries ([247a0f7](https://github.com/matt-riley/mjrwtf/commit/247a0f7c7fac755064040ea78a4094348f2c6b61))
* **main:** add Tailscale server initialization and shutdown ([daf02c5](https://github.com/matt-riley/mjrwtf/commit/daf02c5e102e2e5d8b914189fb727c5b8e67909d))
* **middleware:** add Tailscale WhoIs authentication middleware ([e02946e](https://github.com/matt-riley/mjrwtf/commit/e02946e52e435298dc9182fff09fc9494ef5c110))
* **server:** add conditional route protection for Tailscale mode ([9486d9b](https://github.com/matt-riley/mjrwtf/commit/9486d9b10a227f280cfcdba749f0ef63a929e6e4))
* **tailscale:** create Tailscale network layer ([ed72ad1](https://github.com/matt-riley/mjrwtf/commit/ed72ad19e11cc871aa9de3b8cca7639c18b17da5))

## [0.9.0](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.8.0...mjrwtf-v0.9.0) (2026-01-02)


### Features

* Add adaptive color support and fix CI workflow ([0bfb9d2](https://github.com/matt-riley/mjrwtf/commit/0bfb9d2951352b0eb3ffd91f393ab7d056aff345))
* Add centralized TUI style infrastructure with Catppuccin Mocha palette ([9683f06](https://github.com/matt-riley/mjrwtf/commit/9683f06e8d4fd80ebb96e5b1c2f63ee0f243f05e))
* **tui:** Apply centralized styles to header, footer, and status bar (issue [#213](https://github.com/matt-riley/mjrwtf/issues/213)) ([1f9466d](https://github.com/matt-riley/mjrwtf/commit/1f9466d12837cbfeb0e295411728d91ca324cacc))
* **tui:** style form and modal views ([#215](https://github.com/matt-riley/mjrwtf/issues/215)) ([45288fc](https://github.com/matt-riley/mjrwtf/commit/45288fce6bfb885e5928cec3f0667d8c79f54b16))


### Bug Fixes

* **tui:** Make status prefix matching case-insensitive ([9820d58](https://github.com/matt-riley/mjrwtf/commit/9820d58f2ae8604cc46c70b9f4cf1aec184ecdf5))
* **tui:** preserve status colors ([4374f76](https://github.com/matt-riley/mjrwtf/commit/4374f76067863f7683d5766b922a82b745f945d9))

## [0.8.0](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.7.0...mjrwtf-v0.8.0) (2026-01-02)


### Features

* Add visual assets infrastructure and TUI demo integration ([6a873c3](https://github.com/matt-riley/mjrwtf/commit/6a873c3e196c8004e95377dbd50329da022f40d2))

## [0.7.0](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.6.0...mjrwtf-v0.7.0) (2026-01-01)


### Features

* **tui:** add Bubble Tea skeleton and config loading ([1668fa6](https://github.com/matt-riley/mjrwtf/commit/1668fa607a1e7375085d159cf94adcc2b335b58a))

## [0.6.0](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.5.1...mjrwtf-v0.6.0) (2026-01-01)


### Features

* **tui:** add internal HTTP client ([31206d0](https://github.com/matt-riley/mjrwtf/commit/31206d09f9358eb178e138272b7c8d39fd758eaf))

## [0.5.1](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.5.0...mjrwtf-v0.5.1) (2026-01-01)


### Bug Fixes

* **deps:** update module github.com/mattn/go-sqlite3 to v1.14.33 ([2415f38](https://github.com/matt-riley/mjrwtf/commit/2415f38e9968e63fa8c1bb7be78ce13b07def87d))

## [0.5.0](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.4.1...mjrwtf-v0.5.0) (2026-01-01)


### Features

* **docker:** add migration binary and entrypoint script to run DB migrations before server start ([b91b9cc](https://github.com/matt-riley/mjrwtf/commit/b91b9cc91dca394b4a3a249c6d06a6dad4ca4e5b))

## [0.4.1](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.4.0...mjrwtf-v0.4.1) (2026-01-01)


### Bug Fixes

* **deps:** update module github.com/a-h/templ to v0.3.977 ([88c4b82](https://github.com/matt-riley/mjrwtf/commit/88c4b820b431897fb6744b5d873d08d2b4b15515))

## [0.4.0](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.3.0...mjrwtf-v0.4.0) (2025-12-31)


### Features

* periodic destination checks + gone interstitial (404/410) ([a8839a8](https://github.com/matt-riley/mjrwtf/commit/a8839a8d991fbfeeefd30cc95b746d360b282f04))
* periodic URL status checks and gone interstitial ([d6b3de4](https://github.com/matt-riley/mjrwtf/commit/d6b3de4a0fc4c77f9d63485cf97e86d6bd302ce8))


### Bug Fixes

* address PR review comments ([ed8d008](https://github.com/matt-riley/mjrwtf/commit/ed8d008a1d6261e68ad5e7aeca040622178963e8))
* address review feedback ([3ff3203](https://github.com/matt-riley/mjrwtf/commit/3ff32034b2fb3f111069213673a452fc641ffd34))

## [0.3.0](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.2.0...mjrwtf-v0.3.0) (2025-12-29)


### Features

* **auth:** support AUTH_TOKENS rotation ([b1c416a](https://github.com/matt-riley/mjrwtf/commit/b1c416a335991c0925314c13cc6762cb794fec82))


### Bug Fixes

* constant-time guard for empty token lists ([62c2e40](https://github.com/matt-riley/mjrwtf/commit/62c2e40bf62e20018041e5953a0d9af87ca51fe9))
* fail closed on empty auth token list ([535f9a7](https://github.com/matt-riley/mjrwtf/commit/535f9a7a1be13521803f741f0567d7e87370f2ec))

## [0.2.0](https://github.com/matt-riley/mjrwtf/compare/mjrwtf-v0.1.0...mjrwtf-v0.2.0) (2025-12-28)


### Features

* Add Prometheus metrics integration ([4642ddb](https://github.com/matt-riley/mjrwtf/commit/4642ddb35f52bab98e8344316f302800c7e08e76))
* add some agent skills ([fdc9475](https://github.com/matt-riley/mjrwtf/commit/fdc9475ffca28ba4710653f4b7860a8e535b9e49))
* implement geolocation service integration ([86bf2e5](https://github.com/matt-riley/mjrwtf/commit/86bf2e5fc65dd7445bcfd35d9259b93e43d98227))
* implement http.Flusher, http.Hijacker, and http.Pusher for metricsResponseWriter ([ec10689](https://github.com/matt-riley/mjrwtf/commit/ec1068973616eb9de3d661343283f6e44e52efa0))
* initial project setup ([c70859a](https://github.com/matt-riley/mjrwtf/commit/c70859a8e26a0c15d1d17d8608b0d406eedc94e8))


### Bug Fixes

* agents ([0665dee](https://github.com/matt-riley/mjrwtf/commit/0665dee18cd721b6874733c1cc888fe66b11b64f))
* **deps:** update module github.com/oschwald/geoip2-golang to v2 ([299ecc0](https://github.com/matt-riley/mjrwtf/commit/299ecc09d5c8969ed732be6ae6713e3e1a351cfb))
* **deps:** update module github.com/oschwald/geoip2-golang to v2 ([380e1e8](https://github.com/matt-riley/mjrwtf/commit/380e1e8485827a7536f4bf378f50599956729a6e))
* **deps:** update module github.com/oschwald/geoip2-golang to v2 ([0749a4a](https://github.com/matt-riley/mjrwtf/commit/0749a4a7efc6b3f2ea52c8f71e841e2ea8b657a6))
* **deps:** update module github.com/oschwald/geoip2-golang to v2 ([3a15014](https://github.com/matt-riley/mjrwtf/commit/3a150140095123a50269643f401637574db45124))
* **deps:** update module github.com/oschwald/geoip2-golang to v2 ([dd0c5ec](https://github.com/matt-riley/mjrwtf/commit/dd0c5ec147a07f5f4f4bf2d50c5eca55faaec9ee))
* **deps:** update module github.com/oschwald/geoip2-golang to v2 ([eb46afa](https://github.com/matt-riley/mjrwtf/commit/eb46afafc02b2fb54ebd93bccbc7bb86ba6a174c))
* **deps:** update module github.com/oschwald/geoip2-golang/v2 to v2.1.0 ([981415c](https://github.com/matt-riley/mjrwtf/commit/981415c7105212153cdc9d923ea1d8a601b4f7eb))

## Changelog

This project’s changelog is generated automatically by Release Please.

- Release PRs update this file based on conventional commits.
- GitHub Releases include attached `server` + `migrate` binaries (and checksums).
