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
