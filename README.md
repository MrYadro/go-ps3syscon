# go-ps3syscon

[![Publish Release](https://github.com/MrYadro/go-ps3syscon/actions/workflows/release.yaml/badge.svg)](https://github.com/MrYadro/go-ps3syscon/actions/workflows/release.yaml) [![Build](https://github.com/MrYadro/go-ps3syscon/actions/workflows/build.yaml/badge.svg)](https://github.com/MrYadro/go-ps3syscon/actions/workflows/build.yaml) [![CodeQL](https://github.com/MrYadro/go-ps3syscon/actions/workflows/codeql-analysis.yaml/badge.svg)](https://github.com/MrYadro/go-ps3syscon/actions/workflows/codeql-analysis.yaml) [![report card](https://goreportcard.com/badge/github.com/MrYadro/go-ps3syscon)](https://goreportcard.com/report/github.com/MrYadro/go-ps3syscon) [![releases](https://img.shields.io/github/downloads-pre/MrYadro/go-ps3syscon/latest/total)](https://github.com/MrYadro/go-ps3syscon/releases) [![version](https://img.shields.io/github/v/release/MrYadro/go-ps3syscon?include_prereleases)](https://github.com/MrYadro/go-ps3syscon/releases)

Golang implementation of syscon for PlayStation 3

You can provide `-port`, `-mode` and `-noverify`  params when running binary

`-port` is your USB to TTL port

`-mode` for now supports only `cxrf` and `cxr` mode

`-noverify` don't verify passed commands

List of virtual commands to use:
* `auth` - authorises to use other commands
* `errinfo 0xa0093003` - prints info about `0xa0093003` error `Fatal booting error on step 09 with error info: POWER FAIL`
* `cmdinfo becount` - prints info about `becount` command `becount - Display bringup/shutdown count + Power-on time, command called with no parametres and no subcommands`