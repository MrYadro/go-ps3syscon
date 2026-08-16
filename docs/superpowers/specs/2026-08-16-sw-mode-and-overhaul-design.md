# go-ps3syscon: SW Mode + Full Overhaul — Design

Date: 2026-08-16
Status: Approved in brainstorming (all four design sections)

## Summary

Overhaul go-ps3syscon into a library-first CLI with three fully working syscon
service modes — CXR (external), CXRF (internal), and SW (Sherwood/SuperSlim,
currently unimplemented) — plus a redesigned CLI, session logging, and a test
suite that verifies all protocol behavior without hardware, against the
reference Python implementation (db260179/ps3syscon `Linux/ps3_syscon_uart_script.py`).

The tool remains an interactive console for talking to a PS3 syscon over UART.
It does not dump firmware to files and does not analyze firmware binaries.

## Goals

1. Implement SW mode end-to-end (framing, multi-line parsing, SETCMDLONG, auth).
2. Restructure the codebase into clean, testable packages.
3. Fix known protocol bugs (CXR return-code radix, crash-on-read-error, typo'd API).
4. Redesign the CLI (flags, port discovery, session logging, batch execution).
5. Rewrite the README for the new interface.

## Non-goals

- Firmware image analysis or disassembly of any kind.
- Writing/syscon reflash support beyond what exists today.
- GUI. Plugin/extension framework. Config files.
- Behavior changes to errinfo/cmdinfo data tables (content preserved as-is).

## Architecture

```
go-ps3syscon/
├── cmd/go-ps3syscon/main.go      # thin: flags -> open port -> run console
├── internal/protocol/
│   ├── protocol.go               # Conn, Mode enum, Command() dispatch
│   ├── cxr.go                    # CXR framing/parsing
│   ├── cxrf.go                   # CXRF framing/parsing
│   ├── sw.go                     # SW framing/parsing
│   ├── auth.go                   # AUTH1/AUTH2 AES flow (all modes)
│   └── serial.go                 # go.bug.st/serial adapter -> io.ReadWriteCloser
├── internal/console/
│   ├── console.go                # REPL loop, readline setup
│   ├── commands.go               # command/error tables + autocompletion
│   └── format.go                 # mode-appropriate output formatting, log hook
└── internal/protocol/*_test.go   # unit tests with fake port
```

Dependency rules:

- `internal/protocol` imports only the standard library and depends on
  `io.ReadWriteCloser` for transport. The serial library appears only in
  `serial.go` as an adapter.
- `internal/console` depends on `internal/protocol` (through a small interface
  so console tests can fake it) and on readline.
- `cmd/go-ps3syscon` wires flags, serial adapter, protocol, and console.

## Protocol layer

### Common types

```go
type Mode int // ModeCXR, ModeCXRF, ModeSW

type Conn struct {
    RW      io.ReadWriteCloser
    Mode    Mode
    Timeout time.Duration // read deadline; default 1s
}

type Result struct {
    Code  uint32   // hex status code where the mode defines one; 0 = OK
    Lines []string // payload lines/tokens, mode-dependent
    Raw   string   // raw response text (CXRF full text; others decoded payload)
}

func (c *Conn) Command(cmd string) (Result, error)
```

Error semantics: transport/framing failures (checksum mismatch, malformed
frame, timeout) are returned as `error`; device-reported status (nonzero hex
codes) is returned in `Result.Code`. The REPL prints either and never exits on
a protocol error.

### Baud rates

CXR: 57600. CXRF: 115200. SW: 57600. (Matches reference; not user-configurable.)

### CXR (external service mode)

- Send: `C:{csum:02X}:{cmd}` written in 15-byte chunks, terminating `\r\n`.
  The full formatted string is split on 15-byte boundaries — byte-identical to
  the reference implementation.
- Checksum: 8-bit sum of ASCII bytes of the payload portion (`cmd` on send,
  `data` on receive), formatted `%02X`.
- Receive frame: `R|E:{csum}:{data}` — verify magic, verify checksum.
- `data` splits on spaces: `[OK, code, payload...]`. `Code` is parsed as
  **hex** (`int(x, 16)` in the reference; the current Go code parses decimal —
  this is a bug being fixed).
- `E` frames carry an error code with no payload. `R` frames with a non-OK
  status also return the code with empty payload.
- Receive accumulates bytes until a deadline with no new data or a complete
  frame, rather than a single blocking read.

### CXRF (internal service mode)

- Send: `{cmd}\r\n`. Receive: accumulate until `\r\n`, drop the first line
  (local echo), return the remainder trimmed — same observable behavior as
  today, implemented with read deadlines instead of unbounded reads.

### SW (Sherwood syscon, new)

- Send: `{cmd}:{csum:02X}\r\n` — checksum **suffixed**, no `C:` prefix, no
  chunking.
- Long commands (`len(cmd) >= 0x40`) require sending `SETCMDLONG FF FF` first;
  if its return code is nonzero, abort with that error.
- Receive: multi-line response; **every** line ends with `:{csum}` and every
  line's checksum (over the bytes before the final colon) is verified.
- The last line's space-split tokens: `[cmdEcho, code, payload...]`. `code` is
  all-hex-digits and parsed as hex. Earlier lines are payload lines.
- Result: `Code` from the last line; `Lines` = payload lines (all but last) if
  any, else the last line's payload tokens.

### Auth

One shared implementation in `auth.go`:

- AES-128-CBC with the existing keys/IV/constants (sc2TBKey, tb2SCKey,
  auth1Response, headers, zero IV) — moved from `utils.go` unchanged in value.
- Challenge/response logic (decrypt AUTH1 body, validate the known plaintext,
  swap halves, re-encrypt, emit AUTH2 hex) becomes a pure function testable
  with fixed vectors.
- CXRF variant wraps the flow in `scopen` → expect `SC_READY` before and
  `SC_SUCCESS` after; CXR and SW send `AUTH1 <hex>` / `AUTH2 <hex>` directly.
- Console `auth` built-in dispatches per mode; SW uses the CXR flow with SW
  framing.

### Bug fixes folded in

- `proccessCommand` → `Command` (typo).
- CXR status codes parsed hex, not decimal.
- `log.Fatal` on serial read errors replaced by error returns to the REPL.
- Fixed 1-second sleep between send and receive replaced by deadline-based
  receive (configurable `Timeout`, default 1s).
- Copy-pasted CXRF/CXR auth logic unified.
- Global mutable `cmdList`/completer state removed (constructed per mode).

## CLI design

Flags:

- `-port <dev>` — required for a session; no default (the current default is
  one developer's personal device path).
- `-list-ports` — print detected serial ports and exit (go.bug.st enumeration);
  takes precedence over all other flags.
- `-mode cxr|cxrf|sw` — default `cxrf`.
- `-l <file>` — append all session output (commands and results) to a log file.
- `-e "<command>"` — repeatable; run the given commands and exit. Enables
  scripting and end-to-end tests with a fake port.
- `-v` — verbose: byte counts, raw frames (replaces always-on prints).

REPL (default when no `-e`): readline with mode-aware tab completion
(`intCmd` table for cxrf, `extCmd` for cxr/sw), built-ins `help`, `auth`,
`errinfo <code>`, `cmdinfo <cmd>`, `quit`.

Output formatting (`console/format.go`):

- CXR: `{CODE:08X} {payload tokens}`
- SW: `{CODE:08X}` then payload lines
- CXRF: raw response text

All formatted output additionally appended to the `-l` log file when set.

## Testing

No hardware is available during development; correctness is established by:

1. **Framing/parsing unit tests** — canned byte-stream transcripts fed through
   a fake `io.ReadWriteCloser`:
   - CXR: happy path, checksum mismatch, error frames, chunked-send byte
     layout of long commands.
   - CXRF: echo + response, multi-read accumulation.
   - SW: single-line, multi-line with per-line checksums, bad line checksum,
     long command triggering SETCMDLONG, SETCMDLONG failure.
2. **Crypto tests** — fixed-vector AUTH1 challenge in, exact AUTH2 bytes out.
3. **Console tests** — REPL against a fake protocol: auth, errinfo, unknown
   command, `-e` batch execution, log file contents.
4. **Reference annotation rule** — every behavioral test carries a comment
   citing the corresponding lines/logic in the reference Python script.

CI: existing build/CodeQL workflows plus `go test ./...` (added to the build
workflow if not already present). `gofmt`/`go vet` clean.

## Documentation

README rewritten: what the tool is (UART console), mode table (CXR external /
CXRF internal / SW Sherwood-SuperSlim with hardware caveats from the
reference), new flags with examples (`-list-ports`, `-l`, `-e`), install
instructions unchanged, links to db260179 guides and PS3DevWiki.

## Migration notes

- Binary name and `go install` path unchanged (`cmd/go-ps3syscon`).
- Data tables (errors, command descriptions) move verbatim into
  `internal/console/commands.go`.
- Release workflow untouched.
