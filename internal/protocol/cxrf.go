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

// readUntilCRLF accumulates reads until a complete CRLF-terminated line
// exists AFTER the first line. The echoed command line itself ends with
// CRLF, so it must not satisfy the condition — only a CRLF-terminated
// response line following the echo completes the exchange. Reads that
// error (deadline, EOF) or return no bytes end accumulation.
func (c *Conn) readUntilCRLF() []byte {
	var data []byte
	chunk := make([]byte, 1024)
	for {
		s := string(data)
		idx := strings.IndexByte(s, '\n')
		if idx >= 0 && strings.HasSuffix(s[idx+1:], "\r\n") {
			return data
		}
		n, err := c.RW.Read(chunk)
		data = append(data, chunk[:n]...)
		if err != nil {
			return data
		}
		if n == 0 {
			return data
		}
	}
}

// parseCXRF drops the echoed first line and trims the remainder.
func parseCXRF(data []byte) (Result, error) {
	s := strings.SplitAfterN(string(data), "\n", 2)
	if len(s) < 2 {
		return Result{}, errors.New("cxrf: wrong response")
	}
	return Result{Raw: strings.TrimSpace(s[1])}, nil
}
