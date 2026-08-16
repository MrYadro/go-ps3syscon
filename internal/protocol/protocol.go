// Package protocol implements the PS3 syscon service-mode UART protocols:
// CXR (external), CXRF (internal) and SW (Sherwood).
package protocol

import (
	"bytes"
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
	for i := range len(s) {
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
// errors (deadline, EOF) or returns no bytes ends accumulation. Per the
// spec, a buffer that already parses as a complete CXR frame also ends
// accumulation, so one Command consumes exactly one response frame. With
// no bytes at all, the error is returned. A cheap incremental precheck
// (frame magic plus a running colon count) avoids re-attempting the full
// parse on every chunk.
func (c *Conn) readUntilIdle() ([]byte, error) {
	var data []byte
	chunk := make([]byte, 1024)
	colons := 0
	for {
		n, err := c.RW.Read(chunk)
		data = append(data, chunk[:n]...)
		colons += bytes.Count(chunk[:n], colon)
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
		// Cheap incremental precheck before the full parse: a CXR frame
		// has exactly three colon-separated parts and the R|E: magic
		// after leading whitespace. Colons never disappear, so once
		// the count passes 2 the buffer can never parse.
		if colons == 2 && hasCXRMagic(data) {
			if _, perr := parseCXR(string(data)); perr == nil {
				return data, nil
			}
		}
	}
}

var colon = []byte(":")

// hasCXRMagic reports whether data starts, after leading ASCII
// whitespace, with the "R:" or "E:" frame magic — a necessary condition
// for parseCXR to succeed.
func hasCXRMagic(data []byte) bool {
	i := 0
	for i < len(data) {
		switch data[i] {
		case ' ', '\t', '\r', '\n', '\v', '\f':
			i++
			continue
		}
		break
	}
	return i+1 < len(data) && (data[i] == 'R' || data[i] == 'E') && data[i+1] == ':'
}
