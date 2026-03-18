# bundle-install

## `gcr.io/paketo-buildpacks/bundle-install`

A Cloud Native Buildpack to install gems from a Gemfile


This will be providing `gems`.

## Build Targets

The buildpack binaries are compiled for `linux/amd64` and `linux/arm64` by default.

To build for custom OS/architecture targets, run [scripts/build.sh](scripts/build.sh) with one or more `--target` flags:

```bash
./scripts/build.sh --target linux/amd64 --target linux/arm64
```
