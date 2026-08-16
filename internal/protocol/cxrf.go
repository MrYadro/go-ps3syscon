package protocol

import (
	"bytes"
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
// error (deadline, EOF) or return no bytes end accumulation. Scanning is
// incremental: only newly appended bytes are inspected, so cost grows
// linearly across reads.
func (c *Conn) readUntilCRLF() []byte {
	var data []byte
	chunk := make([]byte, 1024)
	scanned := 0 // data[:scanned] has already been searched for the first '\n'
	nl := -1     // offset of the first '\n', once seen
	for {
		if nl < 0 {
			if i := bytes.IndexByte(data[scanned:], '\n'); i >= 0 {
				nl = scanned + i
			} else {
				scanned = len(data)
			}
		}
		// Complete when the buffer ends with CRLF placed after the
		// first '\n' (equivalent to HasSuffix(s[nl+1:], "\r\n")).
		if nl >= 0 && len(data) >= nl+3 && data[len(data)-2] == '\r' && data[len(data)-1] == '\n' {
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
