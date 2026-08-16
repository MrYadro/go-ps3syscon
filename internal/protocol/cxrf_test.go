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

// On real hardware the echoed command arrives in its own Read before the
// response: readUntilCRLF must not stop on the echo's CRLF.
func TestCXRFCommandEchoSeparateRead(t *testing.T) {
	f := newFakePort([]byte("version\r\n"), []byte("SCM-4.4.2\r\n"))
	c := NewConn(f, ModeCXRF)
	res, err := c.Command("version")
	if err != nil {
		t.Fatal(err)
	}
	if res.Raw != "SCM-4.4.2" {
		t.Fatalf("Raw = %q, want SCM-4.4.2", res.Raw)
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
		t.Fatalf("Raw = %q, want SCM-3.0", res.Raw)
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
		t.Fatalf("Raw = %q, want line1\\r\\nline2", res.Raw)
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
