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
