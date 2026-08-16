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
