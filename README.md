# go-ps3syscon Syscon for PlayStation 3

[![Publish Release](https://github.com/MrYadro/go-ps3syscon/actions/workflows/release.yml/badge.svg)](https://github.com/MrYadro/go-ps3syscon/actions/workflows/release.yml) [![Build](https://github.com/MrYadro/go-ps3syscon/actions/workflows/build.yml/badge.svg)](https://github.com/MrYadro/go-ps3syscon/actions/workflows/build.yml) [![CodeQL](https://github.com/MrYadro/go-ps3syscon/actions/workflows/codeql.yml/badge.svg)](https://github.com/MrYadro/go-ps3syscon/actions/workflows/codeql.yml) [![report card](https://goreportcard.com/badge/github.com/MrYadro/go-ps3syscon)](https://goreportcard.com/report/github.com/MrYadro/go-ps3syscon) [![releases](https://img.shields.io/github/downloads-pre/MrYadro/go-ps3syscon/latest/total)](https://github.com/MrYadro/go-ps3syscon/releases) [![version](https://img.shields.io/github/v/release/MrYadro/go-ps3syscon?include_prereleases)](https://github.com/MrYadro/go-ps3syscon/releases)

![syscon preview](https://user-images.githubusercontent.com/1587606/212574878-ea04b9ab-2da7-4873-91e6-45992a43d59e.png)

## Building and releases

You can download pre build binaries from releases

If you want to build it yourself use (you sould have golang installed)

`go install github.com/MrYadro/go-ps3syscon/cmd/go-ps3syscon@latest`

## How to use it

You can provide `-port` and `-mode`  params when running binary

`-port` is your USB to TTL port

`-mode` for now supports only `cxrf` and `cxr` mode

Every command has it's autocompleteon so use tab more often

List of non-standart commands to use:
* `auth` - authorises to use other commands
* `errinfo 0xa0093003` - prints info about `0xa0093003` error `Fatal booting error on step 09 with error info: POWER FAIL`
* `cmdinfo becount` - prints info about `becount` command `becount - Display bringup/shutdown count + Power-on time, command called with no parametres and no subcommands`

## Links

A PS3 Story: The Yellow Light Of Death: https://www.youtube.com/watch?v=I0UMG3iVYZI

Mostly based on: https://github.com/db260179/ps3syscon