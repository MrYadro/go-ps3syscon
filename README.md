# go-ps3syscon

![publish release](https://github.com/MrYadro/go-ps3syscon/actions/workflows/release.yaml/badge.svg) ![report card](https://goreportcard.com/badge/github.com/MrYadro/go-ps3syscon) ![downloads](https://img.shields.io/github/downloads-pre/MrYadro/go-ps3syscon/latest/total) ![version](https://img.shields.io/github/v/release/MrYadro/go-ps3syscon?include_prereleases)

go-ps3syscon is golang implementation of syscon for PlayStation 3

You can provide `-port` and `-mode` params when running binary

`-mode` for now supports only `cxrf` mode

List of virtual commands to use:
* `auth` - authorises to use other commands
* `errinfo 0xa0093003` - prints info about `0xa0093003` error