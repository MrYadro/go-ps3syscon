# go-ps3syscon — Syscon console for PlayStation 3

[![Publish Release](https://github.com/MrYadro/go-ps3syscon/actions/workflows/release.yml/badge.svg)](https://github.com/MrYadro/go-ps3syscon/actions/workflows/release.yml) [![Build](https://github.com/MrYadro/go-ps3syscon/actions/workflows/build.yml/badge.svg)](https://github.com/MrYadro/go-ps3syscon/actions/workflows/build.yml) [![report card](https://goreportcard.com/badge/github.com/MrYadro/go-ps3syscon)](https://goreportcard.com/report/github.com/MrYadro/go-ps3syscon) [![releases](https://img.shields.io/github/downloads-pre/MrYadro/go-ps3syscon/latest/total)](https://github.com/MrYadro/go-ps3syscon/releases) [![version](https://img.shields.io/github/v/release/MrYadro/go-ps3syscon?include_prereleases)](https://github.com/MrYadro/go-ps3syscon/releases)

![syscon preview](https://user-images.githubusercontent.com/1587606/212574878-ea04b9ab-2da7-4873-91e6-45992a43d59e.png)

Interactive console for talking to a PS3 syscon over a USB-TTL serial
adapter. Based on the guide and tooling from
[db260179/ps3syscon](https://github.com/db260179/ps3syscon).

## Modes

| Mode  | Syscon                              | Baud   | Notes                                  |
|-------|-------------------------------------|--------|----------------------------------------|
| cxrf  | internal service mode               | 115200 | default                                |
| cxr   | external service mode               | 57600  |                                        |
| sw    | Sherwood (SuperSlim)                | 57600  | least tested; not guaranteed on all boards |

## Building and releases

You can download pre-built binaries from
[releases](https://github.com/MrYadro/go-ps3syscon/releases).

To build it yourself (requires Go):

    go install github.com/MrYadro/go-ps3syscon/cmd/go-ps3syscon@latest

## Usage

    go-ps3syscon -port /dev/ttyUSB0 -mode cxrf

Flags:

- `-list-ports` — print detected serial ports and exit
- `-port <dev>` — serial port to use
- `-mode cxr|cxrf|sw` — syscon mode (default cxrf)
- `-l <file>` — append all session output to a log file
- `-e "<command>"` — run a command non-interactively and exit; repeatable,
  e.g. `-e "auth" -e "version"` (scripting/automation)
- `-v` — verbose: print raw frames and byte counts

Every command has tab completion — use tab often.

Built-in REPL commands:

- `auth` — authenticate with the syscon to unlock other commands
- `errinfo 0xa0093003` — prints info about `0xa0093003`: "Fatal booting error
  on step 09 with error info: POWER FAIL"
- `cmdinfo becount` — prints help for the `becount` syscon command
- `help`, `quit`

Everything else is forwarded to the syscon with mode-appropriate framing.

## Development

    go build ./...
    go test ./...
    go vet ./...

The protocol layer (`internal/protocol`) is hardware-independent and tested
against canned response transcripts; behavior mirrors the reference Python
implementation in db260179/ps3syscon.

## Links

- db260179/ps3syscon: https://github.com/db260179/ps3syscon
- PS3DevWiki Syscon Error Codes: https://www.psdevwiki.com/ps3/Syscon_Error_Codes
- A PS3 Story: The Yellow Light Of Death: https://www.youtube.com/watch?v=I0UMG3iVYZI
