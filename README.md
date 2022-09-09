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

On Windows this has to be relative, while on linux it may be absolute.

You can combine download & link, version and destination are required for both. You can configure godl using environment
variables. Environment variables start with GODL_ and then the flag name in all capital and - replaced with _. Boolean
values must be set to true.

Examples:

    GODL_LINK=true
    GODL_VERSION=1.19.1