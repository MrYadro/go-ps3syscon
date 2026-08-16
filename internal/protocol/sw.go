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
	data, err := c.readSW()
	if err != nil {
		return Result{}, err
	}
	return parseSW(string(data))
}

// readSW reads the device's reply in one batch: the reference performs a
// single receive() after a fixed wait per command (no read loop), so each
// SW exchange consumes exactly one read burst.
func (c *Conn) readSW() ([]byte, error) {
	chunk := make([]byte, 4096)
	n, err := c.RW.Read(chunk)
	if n == 0 {
		if err != nil {
			return nil, fmt.Errorf("read: %w", err)
		}
		return nil, fmt.Errorf("read: no data")
	}
	return chunk[:n], nil
}

// swSend writes {cmd}:{csum} — checksum suffixed, no prefix, no chunking.
func (c *Conn) swSend(cmd string) error {
	return c.write([]byte(fmt.Sprintf("%s:%02X\r\n", cmd, countChecksum(cmd))))
}

// parseSW decodes a multi-line response: every line ends with :{csum}
// (verified over the bytes before the LAST colon) and the last line carries
// the hex status code after its leading status-word token.
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
	// The status code is the first hex-parseable token after the leading
	// status word (the reference reads ret[1]; scanning tolerates extra
	// leading tokens on the last line).
	codeIdx := -1
	var code uint64
	for i := 1; i < len(last); i++ {
		if v, err := strconv.ParseUint(last[i], 16, 32); err == nil {
			codeIdx, code = i, v
			break
		}
	}
	if codeIdx < 0 {
		// No status token: whole response is payload, code 0 (reference
		// returns 0 with all lines).
		return Result{Code: 0, Lines: contents, Raw: raw}, nil
	}
	res := Result{Code: uint32(code), Raw: raw}
	if len(contents) == 1 {
		res.Lines = last[codeIdx+1:]
	} else {
		res.Lines = contents[:len(contents)-1]
	}
	return res, nil
}
