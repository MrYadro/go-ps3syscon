# SW Mode + Full Overhaul Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure go-ps3syscon into a library-first CLI with three working syscon modes (CXR, CXRF, SW), a redesigned CLI, session logging, and a hardware-free test suite verified against the reference Python implementation.

**Architecture:** `internal/protocol` (framing/parsing/auth, depends only on `io.ReadWriteCloser` + stdlib, serial lib isolated in `serial.go`), `internal/console` (REPL, command tables, formatting, logging), thin `cmd/go-ps3syscon/main.go` wiring. Tests drive the protocol through a scripted fake port.

**Tech Stack:** Go 1.26, `go.bug.st/serial` v1.8.0, `github.com/chzyer/readline` v1.5.1. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-08-16-sw-mode-and-overhaul-design.md`

## Global Constraints

- Module: `github.com/MrYadro/go-ps3syscon`, Go 1.26. No new dependencies beyond the two already in go.mod.
- `internal/protocol` imports only the standard library — except `serial.go`, the sole file allowed to import `go.bug.st/serial`.
- Protocol behavior must match the reference Python implementation (`db260179/ps3syscon`, `Linux/ps3_syscon_uart_script.py`, `PS3UART.command()` and `.auth()`); every behavioral test cites the reference lines it mirrors.
- No PS3 hardware is available: verification is `go test ./...`, `go build ./...`, `go vet ./...`, `gofmt -l .` (empty output) only.
- Data tables (`scErrors`, `intCmd`, `extCmd`) and crypto constants are moved verbatim — values must not change.
- Error semantics: transport/framing failures (checksum mismatch, malformed frame, timeout) are Go `error`s; device-reported status codes come back in `Result.Code`.
- Baud rates: cxrf 115200; cxr and sw 57600. CXR status codes parse as **hex**. SW commands with `len(cmd) >= 0x40` are preceded by `SETCMDLONG FF FF`.

---

### Task 1: Protocol core (Mode, Result, Conn, checksum, read helper, fake port)

**Files:**
- Create: `internal/protocol/protocol.go`
- Create: `internal/protocol/fake_test.go`
- Test: `internal/protocol/protocol_test.go`

**Interfaces:**
- Consumes: nothing (first task).
- Produces: `type Mode int` with `ModeCXR`, `ModeCXRF`, `ModeSW`; `func ParseMode(string) (Mode, error)`; `func (m Mode) String() string`; `type Result struct { Code uint32; Lines []string; Raw string }`; `type Conn struct { RW io.ReadWriteCloser; Mode Mode; Timeout time.Duration; Verbose bool }`; `func NewConn(rw io.ReadWriteCloser, mode Mode) *Conn` (Timeout defaults to 1s); `func (c *Conn) Command(cmd string) (Result, error)`; unexported `countChecksum(s string) byte`, `c.write([]byte) error`, `c.readUntilIdle() ([]byte, error)`; test fake `newFakePort(reads ...[]byte) *fakePort` with recorded fields `Written []byte` and `Writes [][]byte` (one entry per Write call).

- [ ] **Step 1: Write the failing tests**

`internal/protocol/fake_test.go`:

```go
package protocol

import (
	"io"
	"sync"
)

// fakePort is a scripted io.ReadWriteCloser: each queued []byte is served by
// one Read call; once exhausted, Read reports idle with (0, errIdle).
type fakePort struct {
	mu        sync.Mutex
	responses [][]byte
	readIdx   int
	Written   []byte
	Writes    [][]byte
	Closed    bool
}

var errIdle = &idleError{}

type idleError struct{}

func (*idleError) Error() string { return "fake port idle" }

func newFakePort(reads ...[]byte) *fakePort {
	return &fakePort{responses: reads}
}

func (f *fakePort) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readIdx >= len(f.responses) {
		return 0, errIdle
	}
	n := copy(p, f.responses[f.readIdx])
	f.readIdx++
	return n, nil
}

func (f *fakePort) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Writes = append(f.Writes, append([]byte{}, p...))
	f.Written = append(f.Written, p...)
	return len(p), nil
}

func (f *fakePort) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Closed = true
	return nil
}

var _ io.ReadWriteCloser = (*fakePort)(nil)
```

`internal/protocol/protocol_test.go`:

```go
package protocol

import "testing"

func TestCountChecksum(t *testing.T) {
	// 'A''U''T''H''1' = 65+85+84+72+49 = 355; 355 % 256 = 99 = 0x63
	if got := countChecksum("AUTH1"); got != 0x63 {
		t.Errorf("countChecksum(AUTH1) = %#x, want 0x63", got)
	}
	if got := countChecksum(""); got != 0 {
		t.Errorf(`countChecksum("") = %#x, want 0`, got)
	}
}

func TestReadUntilIdle(t *testing.T) {
	f := newFakePort([]byte("hel"), []byte("lo\n"))
	c := NewConn(f, ModeCXR)
	got, err := c.readUntilIdle()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "hello\n" {
		t.Fatalf("got %q, want %q", got, "hello\n")
	}
}

func TestReadUntilIdleNoData(t *testing.T) {
	c := NewConn(newFakePort(), ModeCXR)
	if _, err := c.readUntilIdle(); err == nil {
		t.Fatal("expected error when port has no data")
	}
}

func TestCommandUnknownMode(t *testing.T) {
	c := &Conn{RW: newFakePort(), Mode: Mode(99)}
	if _, err := c.Command("x"); err == nil {
		t.Fatal("expected error for unknown mode")
	}
}

func TestParseMode(t *testing.T) {
	for in, want := range map[string]Mode{
		"cxr": ModeCXR, "CXR": ModeCXR,
		"cxrf": ModeCXRF,
		"sw": ModeSW, "SW": ModeSW,
	} {
		got, err := ParseMode(in)
		if err != nil || got != want {
			t.Errorf("ParseMode(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseMode("goku"); err == nil {
		t.Error("ParseMode(goku) should fail")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/protocol/`
Expected: FAIL — `undefined: countChecksum`, `undefined: NewConn`, etc.

- [ ] **Step 3: Write the implementation**

`internal/protocol/protocol.go`:

```go
// Package protocol implements the PS3 syscon service-mode UART protocols:
// CXR (external), CXRF (internal) and SW (Sherwood).
package protocol

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Mode identifies a syscon service-mode protocol dialect.
type Mode int

const (
	ModeCXR  Mode = iota // external service mode, 57600 baud
	ModeCXRF             // internal service mode, 115200 baud
	ModeSW               // Sherwood (SuperSlim) service mode, 57600 baud
)

func (m Mode) String() string {
	switch m {
	case ModeCXR:
		return "cxr"
	case ModeCXRF:
		return "cxrf"
	case ModeSW:
		return "sw"
	}
	return "unknown"
}

// ParseMode converts a CLI mode string to a Mode.
func ParseMode(s string) (Mode, error) {
	switch strings.ToLower(s) {
	case "cxr":
		return ModeCXR, nil
	case "cxrf":
		return ModeCXRF, nil
	case "sw":
		return ModeSW, nil
	}
	return 0, fmt.Errorf("unknown mode %q (want cxr, cxrf or sw)", s)
}

// Result is the uniform response of a command across all modes.
type Result struct {
	Code  uint32   // device status code (hex on the wire); 0 = OK
	Lines []string // payload lines/tokens, mode-dependent
	Raw   string   // raw decoded response text
}

// Conn talks to a syscon over any io.ReadWriteCloser.
type Conn struct {
	RW      io.ReadWriteCloser
	Mode    Mode
	Timeout time.Duration // read timeout hint for real ports
	Verbose bool
}

// NewConn wraps rw with defaults for mode.
func NewConn(rw io.ReadWriteCloser, mode Mode) *Conn {
	return &Conn{RW: rw, Mode: mode, Timeout: time.Second}
}

// Command sends cmd and returns the parsed response.
func (c *Conn) Command(cmd string) (Result, error) {
	switch c.Mode {
	case ModeCXR:
		return c.cxrCommand(cmd)
	case ModeCXRF:
		return c.cxrfCommand(cmd)
	case ModeSW:
		return c.swCommand(cmd)
	}
	return Result{}, fmt.Errorf("unknown mode %d", int(c.Mode))
}

// countChecksum is the 8-bit ASCII sum used by every mode.
func countChecksum(s string) byte {
	var sum byte
	for i := 0; i < len(s); i++ {
		sum += s[i]
	}
	return sum
}

func (c *Conn) write(b []byte) error {
	n, err := c.RW.Write(b)
	if c.Verbose {
		fmt.Printf("Sent %d bytes: %q\n", n, b)
	}
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if n != len(b) {
		return fmt.Errorf("short write: %d of %d bytes", n, len(b))
	}
	return nil
}

// readUntilIdle accumulates reads until the port goes idle: a Read that
// errors (deadline, EOF) or returns no bytes ends accumulation. With no
// bytes at all, the error is returned.
func (c *Conn) readUntilIdle() ([]byte, error) {
	var data []byte
	chunk := make([]byte, 1024)
	for {
		n, err := c.RW.Read(chunk)
		data = append(data, chunk[:n]...)
		if err != nil {
			if len(data) > 0 {
				return data, nil
			}
			return nil, fmt.Errorf("read: %w", err)
		}
		if n == 0 {
			if len(data) > 0 {
				return data, nil
			}
			return nil, fmt.Errorf("read: no data")
		}
	}
}
```

The file will not compile yet (`cxrCommand` etc. undefined). Add temporary stubs at the bottom of `protocol.go`, to be replaced by Tasks 2-4:

```go
func (c *Conn) cxrCommand(cmd string) (Result, error)  { return Result{}, nil }
func (c *Conn) cxrfCommand(cmd string) (Result, error) { return Result{}, nil }
func (c *Conn) swCommand(cmd string) (Result, error)   { return Result{}, nil }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/protocol/`
Expected: PASS (all tests)

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: protocol core types, checksum and fake port for tests"
```

---

### Task 2: CXR mode (chunked send, hex response parsing)

**Files:**
- Modify: `internal/protocol/protocol.go` (delete the `cxrCommand` stub)
- Create: `internal/protocol/cxr.go`
- Test: `internal/protocol/cxr_test.go`

**Interfaces:**
- Consumes: `countChecksum`, `c.write`, `c.readUntilIdle`, `fakePort.Writes` from Task 1.
- Produces: `func (c *Conn) cxrCommand(cmd string) (Result, error)`; unexported `func (c *Conn) cxrSend(cmd string) error` and `func parseCXR(resp string) (Result, error)`.

Reference: `ps3_syscon_uart_script.py`, `command()` CXR branch — send loops and `answer` parsing (split on `:`, magic `R`/`E`, checksum verify, `int(x, 16)` status code).

- [ ] **Step 1: Write the failing tests**

`internal/protocol/cxr_test.go`:

```go
package protocol

import (
	"fmt"
	"strings"
	"testing"
)

// cxrFrame builds "R:{csum}:{payload}" — mirrors how the device frames
// responses (reference: answer.split(':') must give exactly 3 parts).
func cxrFrame(payload string) string {
	return fmt.Sprintf("R:%02X:%s", countChecksum(payload), payload)
}

// Reference: commands <= 10 chars are one write of C:{csum}:{cmd}\r\n.
func TestCXRSendShort(t *testing.T) {
	f := newFakePort()
	c := NewConn(f, ModeCXR)
	c.cxrSend("VER") // 'V'+'E'+'R' = 86+69+82 = 237 = 0xED
	want := "C:ED:VER\r\n"
	if string(f.Written) != want {
		t.Fatalf("written %q, want %q", f.Written, want)
	}
	if len(f.Writes) != 1 {
		t.Fatalf("got %d writes, want 1", len(f.Writes))
	}
}

// Reference: for longer commands the first write is C:{csum}:{cmd[:10]},
// then 15-byte payload chunks, remainder + CRLF last.
func TestCXRSendLong(t *testing.T) {
	cmd := strings.Repeat("A", 34) // sum = 34*65 = 2210 % 256 = 0xA2
	f := newFakePort()
	c := NewConn(f, ModeCXR)
	c.cxrSend(cmd)
	if string(f.Written) != "C:A2:"+cmd+"\r\n" {
		t.Fatalf("written %q", f.Written)
	}
	writes := []string{}
	for _, w := range f.Writes {
		writes = append(writes, string(w))
	}
	want := []string{"C:A2:" + cmd[:10], cmd[10:25], cmd[25:] + "\r\n"}
	if len(writes) != len(want) {
		t.Fatalf("writes %q, want %q", writes, want)
	}
	for i := range want {
		if writes[i] != want[i] {
			t.Errorf("write %d = %q, want %q", i, writes[i], want[i])
		}
	}
}

// Reference: length exactly 25 -> first write + one remainder write.
func TestCXRSendBoundary25(t *testing.T) {
	cmd := strings.Repeat("B", 25)
	f := newFakePort()
	c := NewConn(f, ModeCXR)
	c.cxrSend(cmd)
	if len(f.Writes) != 2 {
		t.Fatalf("got %d writes, want 2: %q", len(f.Writes), f.Writes)
	}
	if string(f.Written) != fmt.Sprintf("C:%02X:", countChecksum(cmd))+cmd+"\r\n" {
		t.Fatalf("written %q", f.Written)
	}
}

func TestParseCXR_OK(t *testing.T) {
	res, err := parseCXR(cxrFrame("OK 0 4.2.5"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0 || len(res.Lines) != 1 || res.Lines[0] != "4.2.5" {
		t.Fatalf("got %+v", res)
	}
}

// Reference parses the status code with int(x, 16) — hex, not decimal.
func TestParseCXR_HexCode(t *testing.T) {
	res, err := parseCXR(cxrFrame("OK 10 FFEE"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0x10 {
		t.Fatalf("Code = %#x, want 0x10 (hex radix)", res.Code)
	}
	if len(res.Lines) != 1 || res.Lines[0] != "FFEE" {
		t.Fatalf("Lines = %q", res.Lines)
	}
}

// Reference: non-OK first token -> code only, no payload.
func TestParseCXR_ErrorFrame(t *testing.T) {
	res, err := parseCXR(fmt.Sprintf("E:%02X:NG 12", countChecksum("NG 12")))
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0x12 || len(res.Lines) != 0 {
		t.Fatalf("got %+v", res)
	}
}

func TestParseCXR_BadChecksum(t *testing.T) {
	if _, err := parseCXR("R:00:OK 0 X"); err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestParseCXR_BadMagic(t *testing.T) {
	good := cxrFrame("OK 0 X")
	if _, err := parseCXR("X" + good[1:]); err == nil {
		t.Fatal("expected magic error")
	}
}

func TestParseCXR_TooManyColons(t *testing.T) {
	if _, err := parseCXR(cxrFrame("OK 0 A:B")); err == nil {
		t.Fatal("expected length error (payload colon breaks 3-part split)")
	}
}

func TestParseCXR_BadCode(t *testing.T) {
	if _, err := parseCXR(cxrFrame("OK zz X")); err == nil {
		t.Fatal("expected code parse error")
	}
}

func TestCXRCommandRoundTrip(t *testing.T) {
	f := newFakePort([]byte(cxrFrame("OK 0 SOMEDATA")))
	c := NewConn(f, ModeCXR)
	res, err := c.Command("VER")
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0 || res.Lines[0] != "SOMEDATA" {
		t.Fatalf("got %+v", res)
	}
	if string(f.Written) != "C:ED:VER\r\n" {
		t.Fatalf("written %q", f.Written)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/protocol/ -run CXR -v`
Expected: FAIL — `cxrSend`/`parseCXR` undefined.

- [ ] **Step 3: Write the implementation**

Delete the `cxrCommand` stub from `protocol.go`. Create `internal/protocol/cxr.go`:

```go
package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// cxrCommand implements the external service mode protocol.
// Reference: ps3_syscon_uart_script.py, command(), CXR branch.
func (c *Conn) cxrCommand(cmd string) (Result, error) {
	if err := c.cxrSend(cmd); err != nil {
		return Result{}, err
	}
	data, err := c.readUntilIdle()
	if err != nil {
		return Result{}, err
	}
	return parseCXR(string(data))
}

// cxrSend writes C:{csum}:{cmd}. Commands longer than 10 bytes are split:
// the first write carries the header plus 10 payload bytes, following
// writes carry 15 payload bytes each, and the last write carries the
// remainder plus CRLF (byte-identical to the reference Python).
func (c *Conn) cxrSend(cmd string) error {
	csum := countChecksum(cmd)
	if len(cmd) <= 10 {
		return c.write([]byte(fmt.Sprintf("C:%02X:%s\r\n", csum, cmd)))
	}
	if err := c.write([]byte(fmt.Sprintf("C:%02X:%s", csum, cmd[:10]))); err != nil {
		return err
	}
	j := 10
	for i := len(cmd) - j; i > 15; i -= 15 {
		if err := c.write([]byte(cmd[j : j+15])); err != nil {
			return err
		}
		j += 15
	}
	return c.write([]byte(cmd[j:] + "\r\n"))
}

// parseCXR validates and decodes an R|E:{csum}:{data} frame.
func parseCXR(resp string) (Result, error) {
	resp = strings.TrimSpace(resp)
	parts := strings.Split(resp, ":")
	if len(parts) != 3 {
		return Result{}, errors.New("cxr: wrong response length")
	}
	if parts[0] != "R" && parts[0] != "E" {
		return Result{}, errors.New("cxr: bad magic")
	}
	if parts[1] != fmt.Sprintf("%02X", countChecksum(parts[2])) {
		return Result{}, errors.New("cxr: wrong checksum")
	}
	data := strings.Split(parts[2], " ")
	if parts[0] == "R" && len(data) < 2 || parts[0] == "E" && len(data) != 2 {
		return Result{}, errors.New("cxr: wrong data length")
	}
	// Status code is hexadecimal on the wire (reference: int(x, 16)).
	code, err := strconv.ParseUint(data[1], 16, 32)
	if err != nil {
		return Result{}, fmt.Errorf("cxr: bad status code %q", data[1])
	}
	res := Result{Code: uint32(code), Raw: parts[2]}
	if data[0] == "OK" {
		res.Lines = data[2:]
	}
	return res, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/protocol/ -v`
Expected: PASS (Task 1 + Task 2 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: CXR framing with reference-exact chunking and hex status codes"
```

---

### Task 3: CXRF mode (plain text, echo drop)

**Files:**
- Modify: `internal/protocol/protocol.go` (delete the `cxrfCommand` stub)
- Create: `internal/protocol/cxrf.go`
- Test: `internal/protocol/cxrf_test.go`

**Interfaces:**
- Consumes: `c.write`, `c.RW.Read`, fake port from Task 1.
- Produces: `func (c *Conn) cxrfCommand(cmd string) (Result, error)`; unexported `func (c *Conn) readUntilCRLF() []byte` and `func parseCXRF(data []byte) (Result, error)`.

Reference: current `mode_cxrf.go` receive logic (drop echo line, return rest) and the CXRF branch of `ps3_syscon_uart_script.py` `command()`/`auth()`.

- [ ] **Step 1: Write the failing tests**

`internal/protocol/cxrf_test.go`:

```go
package protocol

import "testing"

func TestCXRFCommand(t *testing.T) {
	f := newFakePort([]byte("version\r\nSCM-4.4.2\r\n"))
	c := NewConn(f, ModeCXRF)
	res, err := c.Command("version")
	if err != nil {
		t.Fatal(err)
	}
	if res.Raw != "SCM-4.4.2" {
		t.Fatalf("Raw = %q, want SCM-4.4.2", res.Raw)
	}
	if string(f.Written) != "version\r\n" {
		t.Fatalf("written %q", f.Written)
	}
}

// Response may arrive split across several reads before the CRLF lands.
func TestCXRFCommandSplitReads(t *testing.T) {
	f := newFakePort([]byte("ve"), []byte("rsion\r"), []byte("\nSCM-3."), []byte("0\r\n"))
	c := NewConn(f, ModeCXRF)
	res, err := c.Command("version")
	if err != nil {
		t.Fatal(err)
	}
	if res.Raw != "SCM-3.0" {
		t.Fatalf("Raw = %q", res.Raw)
	}
}

func TestCXRFMultiLineResponse(t *testing.T) {
	f := newFakePort([]byte("task\r\nline1\r\nline2\r\n"))
	c := NewConn(f, ModeCXRF)
	res, err := c.Command("task")
	if err != nil {
		t.Fatal(err)
	}
	if res.Raw != "line1\r\nline2" {
		t.Fatalf("Raw = %q", res.Raw)
	}
}

func TestCXRFNoResponse(t *testing.T) {
	c := NewConn(newFakePort(), ModeCXRF)
	if _, err := c.Command("version"); err == nil {
		t.Fatal("expected error with no data")
	}
}

func TestCXRFNoCRLF(t *testing.T) {
	f := newFakePort([]byte("version only garbage"))
	c := NewConn(f, ModeCXRF)
	if _, err := c.Command("version"); err == nil {
		t.Fatal("expected error when response never completes")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/protocol/ -run CXRF -v`
Expected: FAIL — stub returns empty Result (Raw mismatch).

- [ ] **Step 3: Write the implementation**

Delete the `cxrfCommand` stub from `protocol.go`. Create `internal/protocol/cxrf.go`:

```go
package protocol

import (
	"errors"
	"strings"
)

// cxrfCommand implements the internal service mode protocol: plain text
// commands; the port echoes the command before the response line.
func (c *Conn) cxrfCommand(cmd string) (Result, error) {
	if err := c.write([]byte(cmd + "\r\n")); err != nil {
		return Result{}, err
	}
	data := c.readUntilCRLF()
	return parseCXRF(data)
}

// readUntilCRLF accumulates reads until the buffer contains a CRLF (the
// echo of our command), matching the old receiveCXRFCommand loop.
func (c *Conn) readUntilCRLF() []byte {
	var data []byte
	chunk := make([]byte, 1024)
	for !strings.Contains(string(data), "\r\n") {
		n, err := c.RW.Read(chunk)
		data = append(data, chunk[:n]...)
		if err != nil {
			return data
		}
		if n == 0 {
			return data
		}
	}
	return data
}

// parseCXRF drops the echoed first line and trims the remainder.
func parseCXRF(data []byte) (Result, error) {
	s := strings.SplitAfterN(string(data), "\n", 2)
	if len(s) < 2 {
		return Result{}, errors.New("cxrf: wrong response")
	}
	return Result{Raw: strings.TrimSpace(s[1])}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/protocol/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: CXRF mode with deadline-safe accumulate-and-echo-drop"
```

---

### Task 4: SW mode (suffixed checksum, SETCMDLONG, multi-line parse)

**Files:**
- Modify: `internal/protocol/protocol.go` (delete the `swCommand` stub)
- Create: `internal/protocol/sw.go`
- Test: `internal/protocol/sw_test.go`

**Interfaces:**
- Consumes: `countChecksum`, `c.write`, `c.readUntilIdle`, `c.Command` from Tasks 1-2.
- Produces: `func (c *Conn) swCommand(cmd string) (Result, error)`; unexported `func (c *Conn) swSend(cmd string) error` and `func parseSW(resp string) (Result, error)`.

Reference: `ps3_syscon_uart_script.py`, `command()` SW branch — `{cmd}:{csum:02X}\r\n` framing, `SETCMDLONG FF FF` for `length >= 0x40`, per-line `rsplit(':', 1)` checksum verification, hex status from the last line's second token.

- [ ] **Step 1: Write the failing tests**

`internal/protocol/sw_test.go`:

```go
package protocol

import (
	"fmt"
	"strings"
	"testing"
)

// swFrame appends the checksum every SW line carries after its last colon.
func swFrame(line string) string {
	return fmt.Sprintf("%s:%02X", line, countChecksum(line))
}

func TestSWSend(t *testing.T) {
	f := newFakePort()
	c := NewConn(f, ModeSW)
	c.swSend("VER") // checksum suffixed, no C: prefix
	want := "VER:" + fmt.Sprintf("%02X", countChecksum("VER")) + "\r\n"
	if string(f.Written) != want {
		t.Fatalf("written %q, want %q", f.Written, want)
	}
}

func TestParseSW_SingleLineWithCode(t *testing.T) {
	res, err := parseSW(swFrame("OK 0 SOMETHING"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0 || len(res.Lines) != 1 || res.Lines[0] != "SOMETHING" {
		t.Fatalf("got %+v", res)
	}
}

func TestParseSW_MultiLine(t *testing.T) {
	resp := strings.Join([]string{
		swFrame("LINE1"),
		swFrame("LINE2"),
		swFrame("LAST OK 12 X"),
	}, "\n")
	res, err := parseSW(resp)
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0x12 {
		t.Fatalf("Code = %#x, want 0x12", res.Code)
	}
	if len(res.Lines) != 2 || res.Lines[0] != "LINE1" || res.Lines[1] != "LINE2" {
		t.Fatalf("Lines = %q", res.Lines)
	}
}

// Reference: checksum is over the bytes before the LAST colon, so payload
// colons are tolerated.
func TestParseSW_PayloadWithColon(t *testing.T) {
	res, err := parseSW(swFrame("OK 0 AA:BB"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines[0] != "AA:BB" {
		t.Fatalf("Lines = %q", res.Lines)
	}
}

func TestParseSW_BadLineChecksum(t *testing.T) {
	resp := swFrame("LINE1") + "\n" + "LINE2:00"
	if _, err := parseSW(resp); err == nil {
		t.Fatal("expected checksum error on second line")
	}
}

func TestParseSW_NoColon(t *testing.T) {
	if _, err := parseSW("NOSEPARATOR"); err == nil {
		t.Fatal("expected format error")
	}
}

// Reference: when the last line has no parseable hex code token, the whole
// response is payload with code 0.
func TestParseSW_NoCode(t *testing.T) {
	res, err := parseSW(swFrame("JUST TEXT"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0 || len(res.Lines) != 1 || res.Lines[0] != "JUST TEXT" {
		t.Fatalf("got %+v", res)
	}
}

func TestSWCommandRoundTrip(t *testing.T) {
	f := newFakePort([]byte(swFrame("OK 0 DATA")))
	c := NewConn(f, ModeSW)
	res, err := c.Command("VER")
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0 || res.Lines[0] != "DATA" {
		t.Fatalf("got %+v", res)
	}
	if string(f.Written) != "VER:"+fmt.Sprintf("%02X", countChecksum("VER"))+"\r\n" {
		t.Fatalf("written %q", f.Written)
	}
}

// Reference: len(cmd) >= 0x40 triggers SETCMDLONG FF FF first; a nonzero
// SETCMDLONG code aborts with 0xFFFFFFFF.
func TestSWCommandLongNeedsSetcmdlong(t *testing.T) {
	long := strings.Repeat("W", 0x40)
	f := newFakePort(
		[]byte(swFrame("OK 0")), // SETCMDLONG response
		[]byte(swFrame("OK 0 PAYLOAD")),
	)
	c := NewConn(f, ModeSW)
	res, err := c.Command(long)
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0 || res.Lines[0] != "PAYLOAD" {
		t.Fatalf("got %+v", res)
	}
	first := string(f.Writes[0])
	if first != "SETCMDLONG FF FF:"+fmt.Sprintf("%02X", countChecksum("SETCMDLONG FF FF"))+"\r\n" {
		t.Fatalf("first write %q", first)
	}
	if len(f.Writes) != 2 {
		t.Fatalf("got %d writes, want 2", len(f.Writes))
	}
}

func TestSWCommandSetcmdlongFails(t *testing.T) {
	long := strings.Repeat("W", 0x40)
	f := newFakePort([]byte(swFrame("NG 1")))
	c := NewConn(f, ModeSW)
	res, err := c.Command(long)
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != 0xFFFFFFFF {
		t.Fatalf("Code = %#x, want 0xFFFFFFFF", res.Code)
	}
	if len(f.Writes) != 1 {
		t.Fatal("long command must not be sent after SETCMDLONG failure")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/protocol/ -run SW -v`
Expected: FAIL — `swSend`/`parseSW` undefined / stub behavior.

- [ ] **Step 3: Write the implementation**

Delete the `swCommand` stub from `protocol.go`. Create `internal/protocol/sw.go`:

```go
package protocol

import (
	"fmt"
	"strconv"
	"strings"
)

// swCommand implements the Sherwood (SuperSlim) service mode protocol.
// Reference: ps3_syscon_uart_script.py, command(), SW branch.
func (c *Conn) swCommand(cmd string) (Result, error) {
	// Commands of 0x40 bytes or more need SETCMDLONG first (reference:
	// `if(length >= 0x40)`). "SETCMDLONG FF FF" is 16 bytes, so no recursion.
	if len(cmd) >= 0x40 {
		res, err := c.Command("SETCMDLONG FF FF")
		if err != nil {
			return Result{}, fmt.Errorf("sw: setcmdlong: %w", err)
		}
		if res.Code != 0 {
			return Result{Code: 0xFFFFFFFF, Raw: "SETCMDLONG failed"}, nil
		}
	}
	if err := c.swSend(cmd); err != nil {
		return Result{}, err
	}
	data, err := c.readUntilIdle()
	if err != nil {
		return Result{}, err
	}
	return parseSW(string(data))
}

// swSend writes {cmd}:{csum} — checksum suffixed, no prefix, no chunking.
func (c *Conn) swSend(cmd string) error {
	return c.write([]byte(fmt.Sprintf("%s:%02X\r\n", cmd, countChecksum(cmd))))
}

// parseSW decodes a multi-line response: every line ends with :{csum}
// (verified over the bytes before the LAST colon) and the last line's
// second space-token is the hex status code.
func parseSW(resp string) (Result, error) {
	lines := strings.Split(strings.TrimSpace(resp), "\n")
	contents := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\n")
		idx := strings.LastIndex(line, ":")
		if idx < 0 {
			return Result{}, fmt.Errorf("sw: wrong response line %q", line)
		}
		body, csum := line[:idx], line[idx+1:]
		if csum != fmt.Sprintf("%02X", countChecksum(body)) {
			return Result{}, fmt.Errorf("sw: wrong checksum on line %q", body)
		}
		contents = append(contents, body)
	}
	if len(contents) == 0 {
		return Result{}, fmt.Errorf("sw: empty response")
	}
	raw := strings.Join(contents, "\n")
	last := strings.Split(contents[len(contents)-1], " ")
	if len(last) < 2 {
		// No status token: whole response is payload, code 0 (reference
		// returns 0 with all lines).
		return Result{Code: 0, Lines: contents, Raw: raw}, nil
	}
	code, err := strconv.ParseUint(last[1], 16, 32)
	if err != nil {
		return Result{Code: 0, Lines: contents, Raw: raw}, nil
	}
	res := Result{Code: uint32(code), Raw: raw}
	if len(contents) == 1 {
		res.Lines = last[2:]
	} else {
		res.Lines = contents[:len(contents)-1]
	}
	return res, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/protocol/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: SW (Sherwood) mode with SETCMDLONG and per-line checksums"
```

---

### Task 5: Auth (shared AES handshake, all modes)

**Files:**
- Create: `internal/protocol/auth.go`
- Test: `internal/protocol/auth_test.go`

**Interfaces:**
- Consumes: `c.Command` from Tasks 2-4.
- Produces: `func (c *Conn) Auth() (string, error)` — the console's `auth` built-in calls this; unexported `func parseAuth1Response(hexResp string) (string, error)`, `decryptCBC`, `encryptCBC`, and the constant `authChallengeHex`.

Reference: current `utils.go` + `mode_cxr.go`/`mode_cxrf.go` auth logic and `ps3_syscon_uart_script.py` `auth()`. Constants move **verbatim** from `cmd/go-ps3syscon/vars.go` lines 5-11.

- [ ] **Step 1: Write the failing tests**

`internal/protocol/auth_test.go`:

```go
package protocol

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

// buildAuth1Response crafts a valid AUTH1 response for a given syscon
// nonce, per the wire format: header(16) || AES-CBC(sc2TBKey, body) where
// body = nonce(8) || zero(8) || auth1Response(16) || zero(16).
func buildAuth1Response(nonce []byte) string {
	body := make([]byte, 0, 0x30)
	body = append(body, nonce...)
	body = append(body, make([]byte, 8)...)
	body = append(body, auth1Response...)
	body = append(body, make([]byte, 16)...)
	resp := append(append([]byte{}, auth1ResponseHeader...), encryptCBC(sc2TBKey, body)...)
	return hex.EncodeToString(resp)
}

func TestParseAuth1ResponseRoundTrip(t *testing.T) {
	nonce := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	auth2Hex, err := parseAuth1Response(buildAuth1Response(nonce))
	if err != nil {
		t.Fatal(err)
	}
	if len(auth2Hex) != 0x80 { // 64 bytes hex-encoded (16 header + 48 body)
		t.Fatalf("len(auth2Hex) = %d, want 128", len(auth2Hex))
	}
	auth2, err := hex.DecodeString(auth2Hex)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(auth2[:0x10], auth2RequestHeader) {
		t.Fatal("AUTH2 header mismatch")
	}
	plain := decryptCBC(tb2SCKey, auth2[0x10:])
	// Expected AUTH2 plaintext: zero(8) || nonce(8) || zero(32) — the
	// reference swaps the nonce into bytes 8..16 of the new body.
	want := append(append(append([]byte{}, make([]byte, 8)...), nonce...), make([]byte, 32)...)
	if !bytes.Equal(plain, want) {
		t.Fatalf("AUTH2 plaintext mismatch:\n got %x\nwant %x", plain, want)
	}
}

func TestParseAuth1Response_BadHeader(t *testing.T) {
	bad := buildAuth1Response([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	flip := "1"
	if string(bad[0]) == "1" {
		flip = "2"
	}
	if _, err := parseAuth1Response(flip + bad[1:]); err == nil {
		t.Fatal("expected header error")
	}
}

func TestParseAuth1Response_BadBody(t *testing.T) {
	b := []byte(buildAuth1Response([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	// Corrupt one hex char in the encrypted body region (hex offset 0x20+).
	b[40]++
	if b[40] > 'f' {
		b[40] = '0'
	}
	if _, err := parseAuth1Response(string(b)); err == nil {
		t.Fatal("expected body error")
	}
}

func TestParseAuth1Response_WrongLength(t *testing.T) {
	if _, err := parseAuth1Response("ABCD"); err == nil {
		t.Fatal("expected length error")
	}
}

func TestParseAuth1Response_BadHex(t *testing.T) {
	if _, err := parseAuth1Response(strings.Repeat("ZZ", 64)); err == nil {
		t.Fatal("expected hex error")
	}
}

func TestCXRAuth(t *testing.T) {
	nonce := []byte{9, 9, 9, 9, 9, 9, 9, 9}
	f := newFakePort(
		[]byte(cxrFrame("OK 0 "+buildAuth1Response(nonce))),
		[]byte(cxrFrame("OK 0")),
	)
	c := NewConn(f, ModeCXR)
	out, err := c.Auth()
	if err != nil {
		t.Fatal(err)
	}
	if out != "Auth successful" {
		t.Fatalf("out = %q", out)
	}
	if !strings.Contains(string(f.Written), "AUTH1 "+authChallengeHex[:10]) {
		t.Fatal("AUTH1 command not sent as expected")
	}
	if !strings.Contains(string(f.Written), "AUTH2 ") {
		t.Fatal("AUTH2 command not sent as expected")
	}
}

func TestCXRAuthFailure(t *testing.T) {
	f := newFakePort(
		[]byte(cxrFrame("OK 0 "+buildAuth1Response([]byte{1, 1, 1, 1, 1, 1, 1, 1}))),
		[]byte(cxrFrame("NG FF")),
	)
	c := NewConn(f, ModeCXR)
	out, err := c.Auth()
	if err != nil {
		t.Fatal(err)
	}
	if out != "Auth failed" {
		t.Fatalf("out = %q, want Auth failed", out)
	}
}

func TestSWAuth(t *testing.T) {
	// SW auth: AUTH1 (134 chars >= 0x40) and AUTH2 (134 chars >= 0x40) each
	// get preceded by SETCMDLONG — 4 scripted responses total.
	resp1 := buildAuth1Response([]byte{7, 7, 7, 7, 7, 7, 7, 7})
	f := newFakePort(
		[]byte(swFrame("OK 0")),
		[]byte(swFrame("OK 0 "+resp1)),
		[]byte(swFrame("OK 0")),
		[]byte(swFrame("OK 0")),
	)
	c := NewConn(f, ModeSW)
	out, err := c.Auth()
	if err != nil {
		t.Fatal(err)
	}
	if out != "Auth successful" {
		t.Fatalf("out = %q", out)
	}
	if len(f.Writes) != 4 {
		t.Fatalf("got %d writes, want 4 (SETCMDLONG+AUTH1+SETCMDLONG+AUTH2)", len(f.Writes))
	}
}

func TestCXRFAuth(t *testing.T) {
	resp1 := buildAuth1Response([]byte{3, 3, 3, 3, 3, 3, 3, 3})
	f := newFakePort(
		[]byte("scopen\r\nSC_READY\r\n"),
		[]byte(authChallengeHex+"\r\n"+resp1+"\r\n"),
		// The AUTH2 command's echo line is dropped by parseCXRF, so any
		// first line works here.
		[]byte("echo\r\nSC_SUCCESS\r\n"),
	)
	c := NewConn(f, ModeCXRF)
	out, err := c.Auth()
	if err != nil {
		t.Fatal(err)
	}
	if out != "Auth successful" {
		t.Fatalf("out = %q", out)
	}
}

func TestCXRFAuthScopenFails(t *testing.T) {
	f := newFakePort([]byte("scopen\r\nSC_BUSY\r\n"))
	c := NewConn(f, ModeCXRF)
	out, err := c.Auth()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Error opening syscon") {
		t.Fatalf("out = %q", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/protocol/ -run 'Auth' -v`
Expected: FAIL — `parseAuth1Response`, `encryptCBC`, constants undefined.

- [ ] **Step 3: Write the implementation**

`internal/protocol/auth.go`:

```go
package protocol

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	// https://www.psdevwiki.com/ps3/Keys
	sc2TBKey            = []byte{113, 240, 63, 24, 76, 1, 197, 235, 195, 246, 162, 42, 66, 186, 149, 37} // 71F03F184C01C5EBC3F6A22A42BA9525
	tb2SCKey            = []byte{144, 126, 115, 15, 77, 78, 10, 11, 123, 117, 240, 48, 235, 29, 157, 54}   // 907E730F4D4E0A0B7B75F030EB1D9D36
	auth1Response       = []byte{51, 80, 189, 120, 32, 52, 92, 41, 5, 106, 34, 59, 162, 32, 179, 35}      // 3350BD7820345C29056A223BA220B323
	zero                = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	auth1ResponseHeader = []byte{16, 16, 0, 0, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0} // 10100000FFFFFFFF0000000000000000
	auth2RequestHeader  = []byte{16, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}          // 10010000000000000000000000000000
)

// authChallengeHex is the fixed AUTH1 challenge body (128 hex chars).
const authChallengeHex = "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

func decryptCBC(key, ciphertext []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err) // keys are fixed 16-byte constants
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, zero).CryptBlocks(plaintext, ciphertext)
	return plaintext
}

func encryptCBC(key, plaintext []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, zero).CryptBlocks(ciphertext, plaintext)
	return ciphertext
}

// parseAuth1Response validates a hex AUTH1 response and returns the hex
// AUTH2 request (header + encrypted body) to send back.
func parseAuth1Response(hexResp string) (string, error) {
	b, err := hex.DecodeString(strings.TrimSpace(hexResp))
	if err != nil {
		return "", fmt.Errorf("auth: bad hex: %w", err)
	}
	if len(b) != 0x40 {
		return "", errors.New("auth: wrong response length")
	}
	if !bytes.Equal(b[0:0x10], auth1ResponseHeader) {
		return "", errors.New("auth: wrong Auth1 response header")
	}
	data := decryptCBC(sc2TBKey, b[0x10:0x40])
	if !bytes.Equal(data[0x8:0x10], zero[0:0x8]) || !bytes.Equal(data[0x10:0x20], auth1Response) || !bytes.Equal(data[0x20:0x30], zero) {
		return "", errors.New("auth: wrong Auth1 response body")
	}
	newData := append(append([]byte{}, data[0x8:0x10]...), data[0x0:0x8]...)
	newData = append(newData, zero...)
	newData = append(newData, zero...)
	body := append(append([]byte{}, auth2RequestHeader...), encryptCBC(tb2SCKey, newData)...)
	return fmt.Sprintf("%02X", body), nil
}

// Auth performs the AUTH1/AUTH2 handshake for the connection's mode.
func (c *Conn) Auth() (string, error) {
	switch c.Mode {
	case ModeCXRF:
		return c.cxrfAuth()
	case ModeCXR, ModeSW:
		return c.cxrAuth()
	}
	return "", fmt.Errorf("auth: unknown mode %d", int(c.Mode))
}

// cxrAuth runs AUTH1/AUTH2 with CXR/SW framing (SW auto-issues SETCMDLONG
// for the long commands inside Command).
func (c *Conn) cxrAuth() (string, error) {
	res, err := c.Command("AUTH1 " + authChallengeHex)
	if err != nil {
		return "", fmt.Errorf("auth1: %w", err)
	}
	if res.Code != 0 || len(res.Lines) < 1 {
		return "", fmt.Errorf("auth1: unexpected response %q", res.Raw)
	}
	auth2, err := parseAuth1Response(res.Lines[0])
	if err != nil {
		return "", err
	}
	res, err = c.Command("AUTH2 " + auth2)
	if err != nil {
		return "", fmt.Errorf("auth2: %w", err)
	}
	if res.Code != 0 {
		return "Auth failed", nil
	}
	return "Auth successful", nil
}

// cxrfAuth wraps the handshake in scopen/SC_SUCCESS.
func (c *Conn) cxrfAuth() (string, error) {
	res, err := c.Command("scopen")
	if err != nil {
		return "", fmt.Errorf("scopen: %w", err)
	}
	if !strings.Contains(res.Raw, "SC_READY") {
		return fmt.Sprintf("Error opening syscon\n%s", res.Raw), nil
	}
	res, err = c.Command(authChallengeHex)
	if err != nil {
		return "", fmt.Errorf("auth1: %w", err)
	}
	auth2, err := parseAuth1Response(res.Raw)
	if err != nil {
		return "", err
	}
	res, err = c.Command(auth2)
	if err != nil {
		return "", fmt.Errorf("auth2: %w", err)
	}
	if !strings.Contains(res.Raw, "SC_SUCCESS") {
		return "Auth failed", nil
	}
	return "Auth successful", nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/protocol/ -v`
Expected: PASS. If `TestParseAuth1Response_BadBody` coincidentally passes validation (AES avalanche makes this near-impossible), change the corrupted offset to hex index 0x30 and re-run.

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: shared AUTH1/AUTH2 handshake for cxr, cxrf and sw modes"
```

---

### Task 6: Serial adapter

**Files:**
- Create: `internal/protocol/serial.go`
- Test: `internal/protocol/serial_test.go`

**Interfaces:**
- Consumes: `Mode` from Task 1.
- Produces: `func OpenSerial(portName string, mode Mode, timeout time.Duration) (io.ReadWriteCloser, error)` — used by `cmd/go-ps3syscon/main.go` (Task 8); unexported `func baudFor(mode Mode) int`.

- [ ] **Step 1: Verify the library API**

Run: `go doc go.bug.st/serial Port.SetReadTimeout`
Expected: documentation for `SetReadTimeout(t time.Duration) error`. If the method does not exist, run `go doc go.bug.st/serial Port` and use whatever timeout-setting method exists, adjusting Step 3 accordingly.

- [ ] **Step 2: Write the failing test**

`internal/protocol/serial_test.go`:

```go
package protocol

import "testing"

func TestBaudFor(t *testing.T) {
	for mode, want := range map[Mode]int{
		ModeCXR:  57600,
		ModeSW:   57600,
		ModeCXRF: 115200,
	} {
		if got := baudFor(mode); got != want {
			t.Errorf("baudFor(%v) = %d, want %d", mode, got, want)
		}
	}
}
```

- [ ] **Step 3: Write the implementation and run the test**

`internal/protocol/serial.go`:

```go
package protocol

import (
	"io"
	"time"

	"go.bug.st/serial"
)

func baudFor(mode Mode) int {
	if mode == ModeCXRF {
		return 115200
	}
	return 57600
}

// OpenSerial opens the named port at the mode's baud rate with a read
// timeout, ready to be wrapped in a Conn. This is the only file in the
// package allowed to import the serial library.
func OpenSerial(portName string, mode Mode, timeout time.Duration) (io.ReadWriteCloser, error) {
	p, err := serial.Open(portName, &serial.Mode{BaudRate: baudFor(mode)})
	if err != nil {
		return nil, err
	}
	if err := p.SetReadTimeout(timeout); err != nil {
		p.Close()
		return nil, err
	}
	p.ResetInputBuffer()
	return p, nil
}
```

Run: `go test ./internal/protocol/ -run BaudFor -v`
Expected: PASS

- [ ] **Step 4: Run the full package suite**

Run: `go test ./... && go vet ./...`
Expected: PASS, no vet issues.

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: serial adapter with per-mode baud and read timeout"
```

---

### Task 7: Console package (REPL, built-ins, tables, formatting, logging)

**Files:**
- Create: `internal/console/console.go`
- Create: `internal/console/commands.go` (data tables, moved verbatim)
- Create: `internal/console/format.go`
- Test: `internal/console/console_test.go`

**Interfaces:**
- Consumes: `protocol.Result`, `protocol.Mode` (+ its constants) from Task 1.
- Produces: `type Commander interface { Command(cmd string) (protocol.Result, error); Auth() (string, error) }` (satisfied by `*protocol.Conn`); `func New(sc Commander, mode protocol.Mode, logFile io.Writer) *Console`; `func (c *Console) HandleLine(line string) (out string, quit bool)`; `func (c *Console) Run() error`; `func (c *Console) Batch(cmds []string)`; unexported `formatResult(mode protocol.Mode, res protocol.Result) string`; `Console.Out io.Writer` field (defaults to os.Stdout; tests override).

- [ ] **Step 1: Move the data tables verbatim**

Create `internal/console/commands.go` with this exact structure:

- Line 1: `package console`
- Line 2: blank
- Line 3 onwards: copy `cmd/go-ps3syscon/vars.go` starting at its line 12 (the blank line right after the `auth = "..."` constant) through the end of the file (the `var ( ... )` wrapper closes over `scErrors`, `extCmd` and `intCmd`), **excluding** the crypto constants on lines 5-11 (`sc2TBKey` through `auth`) — those move to `internal/protocol/auth.go` in Task 5.

Concretely the result is: `package console`, blank, `var (`, blank, `scErrors = map[string]string{`, ... unchanged table content ..., `intCmd = map[...]`, ..., `}` , `)`. Do not reformat, rename, or reorder entries; values are frozen per the spec.

Verify with (zsh):

```zsh
diff <(sed -n '12,$p' cmd/go-ps3syscon/vars.go) <(sed -n '4,$p' internal/console/commands.go)
```

Expected: no output — zero content drift.

- [ ] **Step 2: Write the failing tests**

`internal/console/console_test.go`:

```go
package console

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/MrYadro/go-ps3syscon/internal/protocol"
)

type fakeCommander struct {
	mode    protocol.Mode
	lastCmd string
	result  protocol.Result
	err     error
	authOut string
	authErr error
}

func (f *fakeCommander) Command(cmd string) (protocol.Result, error) {
	f.lastCmd = cmd
	return f.result, f.err
}

func (f *fakeCommander) Auth() (string, error) { return f.authOut, f.authErr }

func testConsole(mode protocol.Mode, fc *fakeCommander) (*Console, *bytes.Buffer) {
	var log bytes.Buffer
	return New(fc, mode, &log), &log
}

func TestHandleLineErrinfo(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, quit := cons.HandleLine("errinfo 0xa0093003")
	if quit {
		t.Fatal("unexpected quit")
	}
	// README example: Fatal booting error on step 09 with error info: POWER FAIL
	if out != "Fatal booting error on step 09 with error info: POWER FAIL" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineErrinfoEmpty(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("errinfo")
	if out != "Please provide error code!" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineErrinfoUnknown(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("errinfo xyz")
	if out != "Unknown error!" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineCmdinfo(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("cmdinfo becount")
	want := "becount - Display bringup/shutdown count + Power-on time, command called with no parametres and no subcommands"
	if out != want {
		t.Fatalf("out = %q, want %q", out, want)
	}
}

func TestHandleLineCmdinfoExtTable(t *testing.T) {
	// cxr/sw use the extCmd table: FAN exists there but not in intCmd.
	cons, _ := testConsole(protocol.ModeCXR, &fakeCommander{})
	out, _ := cons.HandleLine("cmdinfo FAN")
	if strings.HasPrefix(out, "Wrong command") {
		t.Fatalf("FAN should resolve in extCmd table for cxr mode: %q", out)
	}
	cons2, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out2, _ := cons2.HandleLine("cmdinfo FAN")
	if out2 != "Wrong command" {
		t.Fatalf("FAN should not resolve in intCmd table for cxrf mode: %q", out2)
	}
}

func TestHandleLineCmdinfoWrong(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("cmdinfo nosuchcmd")
	if out != "Wrong command" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineAuth(t *testing.T) {
	fc := &fakeCommander{authOut: "Auth successful"}
	cons, _ := testConsole(protocol.ModeCXR, fc)
	out, _ := cons.HandleLine("auth")
	if out != "Auth successful" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineAuthError(t *testing.T) {
	fc := &fakeCommander{authErr: errors.New("boom")}
	cons, _ := testConsole(protocol.ModeCXR, fc)
	out, _ := cons.HandleLine("auth")
	if out != "Error: boom" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLinePassThrough(t *testing.T) {
	fc := &fakeCommander{result: protocol.Result{Code: 0, Lines: []string{"4.4.2"}}}
	cons, _ := testConsole(protocol.ModeCXR, fc)
	out, _ := cons.HandleLine("version")
	if fc.lastCmd != "version" {
		t.Fatalf("lastCmd = %q", fc.lastCmd)
	}
	if out != "00000000 4.4.2" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLinePassThroughError(t *testing.T) {
	fc := &fakeCommander{err: errors.New("cxr: wrong checksum")}
	cons, _ := testConsole(protocol.ModeCXR, fc)
	out, _ := cons.HandleLine("version")
	if out != "Error: cxr: wrong checksum" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineQuit(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXR, &fakeCommander{})
	_, quit := cons.HandleLine("quit")
	if !quit {
		t.Fatal("quit should set quit=true")
	}
}

func TestHandleLineHelp(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("help")
	if !strings.HasPrefix(out, "commands:") {
		t.Fatalf("out = %q", out)
	}
	if !strings.Contains(out, "becount") { // intCmd entry for cxrf
		t.Fatalf("help missing intCmd entries")
	}
}

func TestFormatResult(t *testing.T) {
	if got := formatResult(protocol.ModeCXR, protocol.Result{Code: 0x12, Lines: []string{"A", "B"}}); got != "00000012 A B" {
		t.Fatalf("cxr: %q", got)
	}
	if got := formatResult(protocol.ModeSW, protocol.Result{Code: 0x12, Lines: []string{"A", "B"}}); got != "00000012\nA\nB" {
		t.Fatalf("sw: %q", got)
	}
	if got := formatResult(protocol.ModeSW, protocol.Result{Code: 0x12}); got != "00000012" {
		t.Fatalf("sw empty: %q", got)
	}
	if got := formatResult(protocol.ModeCXRF, protocol.Result{Raw: "SC_READY"}); got != "SC_READY" {
		t.Fatalf("cxrf: %q", got)
	}
}

func TestBatchWritesOutAndLog(t *testing.T) {
	fc := &fakeCommander{result: protocol.Result{Raw: "SCM-4.4.2"}}
	var out bytes.Buffer
	var log bytes.Buffer
	cons := New(fc, protocol.ModeCXRF, &log)
	cons.Out = &out
	cons.Batch([]string{"version", "quit", "auth"})
	if !strings.Contains(out.String(), "> version") || !strings.Contains(out.String(), "SCM-4.4.2") {
		t.Fatalf("batch out = %q", out.String())
	}
	if !strings.Contains(log.String(), "SCM-4.4.2") {
		t.Fatalf("log = %q", log.String())
	}
	if strings.Contains(out.String(), "> auth") {
		t.Fatal("quit must stop batch processing")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/console/`
Expected: FAIL — `New`, `HandleLine`, `formatResult` undefined.

- [ ] **Step 4: Write the implementation**

`internal/console/console.go`:

```go
// Package console implements the interactive syscon REPL and its
// built-in commands on top of the protocol package.
package console

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MrYadro/go-ps3syscon/internal/protocol"
	"github.com/chzyer/readline"
)

// Commander is the subset of protocol.Conn the console needs. Tests fake it.
type Commander interface {
	Command(cmd string) (protocol.Result, error)
	Auth() (string, error)
}

// Console is an interactive syscon session.
type Console struct {
	SC      Commander
	Mode    protocol.Mode
	LogFile io.Writer // optional: results are appended here
	Out     io.Writer // REPL/batch output; defaults to os.Stdout
}

// New creates a Console. logFile may be nil.
func New(sc Commander, mode protocol.Mode, logFile io.Writer) *Console {
	return &Console{SC: sc, Mode: mode, LogFile: logFile, Out: os.Stdout}
}

// HandleLine processes one input line, returning its output and whether
// the session should stop.
func (c *Console) HandleLine(line string) (string, bool) {
	line = strings.TrimSpace(line)
	switch {
	case line == "":
		return "", false
	case line == "quit":
		return "", true
	case line == "help":
		return c.usage(), false
	case strings.HasPrefix(line, "auth"):
		out, err := c.SC.Auth()
		if err != nil {
			return "Error: " + err.Error(), false
		}
		return out, false
	case strings.HasPrefix(line, "errinfo"):
		return c.errinfo(strings.TrimSpace(strings.TrimPrefix(line, "errinfo"))), false
	case strings.HasPrefix(line, "cmdinfo"):
		return c.cmdinfo(strings.TrimSpace(strings.TrimPrefix(line, "cmdinfo"))), false
	default:
		res, err := c.SC.Command(line)
		if err != nil {
			return "Error: " + err.Error(), false
		}
		return formatResult(c.Mode, res), false
	}
}

// Batch runs commands non-interactively, stopping at quit.
func (c *Console) Batch(cmds []string) {
	for _, cmd := range cmds {
		fmt.Fprintf(c.Out, "> %s\n", cmd)
		out, quit := c.HandleLine(cmd)
		if out != "" {
			c.emit(out)
		}
		if quit {
			return
		}
	}
}

// Run starts the interactive REPL.
func (c *Console) Run() error {
	l, err := readline.NewEx(&readline.Config{
		Prompt:            "\033[31mps3syscon>\033[0m ",
		AutoComplete:      newCompleter(c.Mode),
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return err
	}
	defer l.Close()
	l.CaptureExitSignal()
	for {
		line, err := l.Readline()
		if err != nil { // ErrInterrupt or io.EOF (exit prompt)
			return nil
		}
		out, quit := c.HandleLine(line)
		if out != "" {
			c.emit(out)
		}
		if quit {
			return nil
		}
	}
}

func (c *Console) emit(s string) {
	fmt.Fprintln(c.Out, s)
	if c.LogFile != nil {
		fmt.Fprintln(c.LogFile, s)
	}
}

func (c *Console) usage() string {
	return "commands:\n" + newCompleter(c.Mode).Tree("    ")
}

// newCompleter builds tab completion: built-ins plus the mode's command table.
func newCompleter(mode protocol.Mode) *readline.PrefixCompleter {
	table := cmdTable(mode)
	pc := readline.NewPrefixCompleter(
		readline.PcItem("quit"),
		readline.PcItem("help"),
		readline.PcItem("auth"),
		readline.PcItem("errinfo"),
		readline.PcItem("cmdinfo"),
	)
	for name, meta := range table {
		item := readline.PcItem(name)
		if subs, ok := meta["subcommands"]; ok && subs != "" {
			for _, sc := range strings.Split(subs, ",") {
				item.Children = append(item.Children, readline.PcItem(sc))
			}
		}
		pc.Children = append(pc.Children, item)
	}
	return pc
}

func cmdTable(mode protocol.Mode) map[string]map[string]string {
	if mode == protocol.ModeCXRF {
		return intCmd
	}
	return extCmd
}
```

`internal/console/format.go`:

```go
package console

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MrYadro/go-ps3syscon/internal/protocol"
)

// formatResult renders a Result per mode (mirrors the Python tool's output).
func formatResult(mode protocol.Mode, res protocol.Result) string {
	switch mode {
	case protocol.ModeCXR:
		return fmt.Sprintf("%08X %s", res.Code, strings.Join(res.Lines, " "))
	case protocol.ModeSW:
		if len(res.Lines) == 0 {
			return fmt.Sprintf("%08X", res.Code)
		}
		return fmt.Sprintf("%08X\n%s", res.Code, strings.Join(res.Lines, "\n"))
	default: // ModeCXRF
		return res.Raw
	}
}

func (c *Console) errinfo(arg string) string {
	if arg == "" {
		return "Please provide error code!"
	}
	return parseErrorCode(arg)
}

var errCodeRe = regexp.MustCompile(`0xa[A-Fa-f0-9]{3}[1-4][0-9][0-6f][0-9f]`)

func parseErrorCode(err string) string {
	if !errCodeRe.MatchString(err) {
		return "Unknown error!"
	}
	stepNo := err[4:6]
	errCat := err[6:7]
	errNo := err[6:10]
	return fmt.Sprintf("%s on step %s with error info: %s", scErrors[errCat], stepNo, scErrors[errNo])
}

func (c *Console) cmdinfo(arg string) string {
	cm, ok := cmdTable(c.Mode)[arg]
	if !ok {
		return "Wrong command"
	}
	params, ok := cm["parametres"]
	if ok {
		params = strings.Join(strings.Split(params, ","), ", ")
	} else {
		params = "no"
	}
	subcmd, ok := cm["subcommands"]
	if ok {
		subcmd = strings.Join(strings.Split(subcmd, ","), ", ")
	} else {
		subcmd = "no"
	}
	return fmt.Sprintf("%s - %s, command called with %s parametres and %s subcommands", arg, cm["description"], params, subcmd)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/console/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/console/
git commit -m "feat: console package with REPL, built-ins, formatting and logging"
```

---

### Task 8: New main, CLI flags, delete old code

**Files:**
- Create: `cmd/go-ps3syscon/main.go` (replaces all eleven old files in that directory)
- Delete: `cmd/go-ps3syscon/command.go`, `cmd/go-ps3syscon/command_auth.go`, `cmd/go-ps3syscon/command_cmdinfo.go`, `cmd/go-ps3syscon/command_errinfo.go`, `cmd/go-ps3syscon/mode_cxr.go`, `cmd/go-ps3syscon/mode_cxrf.go`, `cmd/go-ps3syscon/mode_sw.go`, `cmd/go-ps3syscon/ps3syscon.go`, `cmd/go-ps3syscon/syscon.go`, `cmd/go-ps3syscon/utils.go`, `cmd/go-ps3syscon/vars.go`

**Interfaces:**
- Consumes: `protocol.ParseMode`, `protocol.OpenSerial`, `protocol.NewConn` (Tasks 1, 6); `console.New`, `Console.Batch`, `Console.Run` (Task 7); `serial.GetPortsList`.
- Produces: the final binary with flags `-port`, `-mode`, `-list-ports`, `-l`, `-e` (repeatable), `-v`.

- [ ] **Step 1: Write main.go**

`cmd/go-ps3syscon/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/MrYadro/go-ps3syscon/internal/console"
	"github.com/MrYadro/go-ps3syscon/internal/protocol"
	"go.bug.st/serial"
)

type cmdList []string

func (l *cmdList) String() string { return strings.Join(*l, "; ") }

func (l *cmdList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

func main() {
	portName := flag.String("port", "", "serial port to use (see -list-ports)")
	modeStr := flag.String("mode", "cxrf", "syscon mode: cxr, cxrf or sw")
	listPorts := flag.Bool("list-ports", false, "list available serial ports and exit")
	logPath := flag.String("l", "", "append session output to this log file")
	verbose := flag.Bool("v", false, "verbose: print raw frames and byte counts")
	var exec cmdList
	flag.Var(&exec, "e", `command to run non-interactively; repeatable, e.g. -e "auth" -e "version"`)
	flag.Parse()

	if *listPorts {
		ports, err := serial.GetPortsList()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error listing ports:", err)
			os.Exit(1)
		}
		for _, p := range ports {
			fmt.Println(p)
		}
		return
	}

	mode, err := protocol.ParseMode(*modeStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *portName == "" {
		fmt.Fprintln(os.Stderr, "no port given: use -port (see -list-ports)")
		os.Exit(1)
	}

	var logWriter io.Writer
	if *logPath != "" {
		logFile, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening log file:", err)
			os.Exit(1)
		}
		defer logFile.Close()
		logWriter = logFile
	}

	rw, err := protocol.OpenSerial(*portName, mode, time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open serial port %s: %v\n", *portName, err)
		os.Exit(1)
	}
	defer rw.Close()

	conn := protocol.NewConn(rw, mode)
	conn.Verbose = *verbose

	cons := console.New(conn, mode, logWriter)
	if len(exec) > 0 {
		cons.Batch(exec)
		return
	}
	if err := cons.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Delete the old implementation files**

```bash
git rm cmd/go-ps3syscon/command.go cmd/go-ps3syscon/command_auth.go \
  cmd/go-ps3syscon/command_cmdinfo.go cmd/go-ps3syscon/command_errinfo.go \
  cmd/go-ps3syscon/mode_cxr.go cmd/go-ps3syscon/mode_cxrf.go \
  cmd/go-ps3syscon/mode_sw.go cmd/go-ps3syscon/ps3syscon.go \
  cmd/go-ps3syscon/syscon.go cmd/go-ps3syscon/utils.go cmd/go-ps3syscon/vars.go
```

- [ ] **Step 3: Build and run all checks**

Run: `go build ./... && go test ./... && go vet ./... && gofmt -l .`
Expected: build OK, all tests PASS, vet clean, gofmt prints nothing.

- [ ] **Step 4: Smoke-test hardware-free CLI paths**

```bash
go run ./cmd/go-ps3syscon -list-ports
go run ./cmd/go-ps3syscon -mode bogus; echo "exit=$?"
go run ./cmd/go-ps3syscon; echo "exit=$?"
```

Expected: first lists ports (possibly zero lines) and exits 0; second prints the unknown-mode error with `exit=1`; third prints the no-port error with `exit=1`.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: redesigned CLI with list-ports, batch mode and session log"
```

---

### Task 9: README rewrite and CI test step

**Files:**
- Modify: `README.md` (full rewrite)
- Modify: `.github/workflows/build.yml`

**Interfaces:**
- Consumes: the finished CLI from Task 8.
- Produces: docs and CI matching the spec's Documentation and CI sections.

- [ ] **Step 1: Rewrite README.md**

Replace the entire file content with (keep the existing badge line and preview image URL from the current README at the top — copy them verbatim from lines 3 and 5 of the old file):

```markdown
# go-ps3syscon — Syscon console for PlayStation 3

[badge line — copy verbatim from old README line 3]

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
```

- [ ] **Step 2: Add test and vet steps to CI**

`.github/workflows/build.yml` — change the final step from:

```yaml
      - run: go build ./cmd/go-ps3syscon
```

to:

```yaml
      - run: go build ./...
      - run: go vet ./...
      - run: go test ./...
```

- [ ] **Step 3: Final verification**

Run: `go build ./... && go test ./... && go vet ./... && gofmt -l .`
Expected: all pass, gofmt silent.

- [ ] **Step 4: Commit**

```bash
git add README.md .github/workflows/build.yml
git commit -m "docs: README and CI for the redesigned CLI with SW mode"
```

---

## Plan Self-Review (already performed)

1. **Spec coverage:** package layout (Tasks 1-8), CXR hex-code bug + chunking (Task 2), CXRF deadline receive (Task 3), SW mode + SETCMDLONG + multi-line checksums (Task 4), shared auth with fixed vectors (Task 5), serial adapter + baud (Task 6), REPL/built-ins/formatting/logging/tables (Task 7), CLI flags incl. `-list-ports`/`-l`/`-e`/`-v` + old-code deletion (Task 8), README + CI `go test` (Task 9). Spec's "reference annotation rule" is satisfied by per-test reference comments.
2. **Placeholders:** none — every step carries exact code or an exact verbatim-copy instruction with source line numbers.
3. **Type consistency:** `Commander` matches `*protocol.Conn` methods (`Command`, `Auth`); `console.New(conn, mode, logFile)` used identically in Task 8; `Result` fields used consistently; fake port `Writes`/`Written` defined in Task 1 and consumed in Tasks 2, 4, 5.




