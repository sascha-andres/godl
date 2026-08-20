# godl

Download go releases

## Usage

You can use the following flags to specifiy actions:

    -print: use to print all versions for current os & arch
    -download: download provided version
    -force-download: force new download
    -skip-download: skip download if it exists (convenience for scripting purposes)
    -link: link go version as linkname
    -link-name: name (path) of symlink, defaulting to current, a link alongside the download location
    -version: download this version
    -verbose: ramp up verbosity
    -destination: save version in this directory
    -tool-version: print the version of godl and the go version it was built with, then exit

On Windows this has to be relative, while on linux it may be absolute.

You can combine download & link, version and destination are required for both. You can configure godl using environment
variables. Environment variables start with GODL_ and then the flag name in all capital and - replaced with _. Boolean
values must be set to true.

Examples:

    GODL_LINK=true
    GODL_VERSION=1.19.1

`godl -tool-version` prints output in the form `godl <version> build with <go version>`,
where `<version>` is the release tag or, for a development build, the commit it was built
from (with a `+dirty` suffix if there were uncommitted changes at build time).

## Development

Every pull request and every push to `main` runs the test suite and `govulncheck`. Pushing a
tag matching `v*` additionally builds and publishes a release (linux/amd64, linux/arm64,
darwin/arm64) via [GoReleaser](https://goreleaser.com).

## History

| Version | Date       | Description     |
| ------- | ---------- | --------------- |
| 0.1.0   | 09/09/2022 | initial release |
