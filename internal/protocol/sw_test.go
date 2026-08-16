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
		swFrame("LAST 12 X"),
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
