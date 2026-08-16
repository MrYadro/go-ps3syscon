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

func (c *Conn) cxrfCommand(cmd string) (Result, error) { return Result{}, nil }
func (c *Conn) swCommand(cmd string) (Result, error)   { return Result{}, nil }
