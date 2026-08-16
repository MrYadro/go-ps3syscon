package protocol

import "testing"

func TestCountChecksum(t *testing.T) {
	// 'A''U''T''H''1' = 65+85+84+72+49 = 355; 355 % 256 = 99 = 0x63
	if got := countChecksum("AUTH1"); got != 0x63 {
		t.Errorf("countChecksum(AUTH1) = %#x, want 0x63", got)
	}
	if got := countChecksum(""); got != 0 {
		t.Errorf(`countChecksum("") = %#x, want 0`, got)
	}
}

func TestReadUntilIdle(t *testing.T) {
	f := newFakePort([]byte("hel"), []byte("lo\n"))
	c := NewConn(f, ModeCXR)
	got, err := c.readUntilIdle()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "hello\n" {
		t.Fatalf("got %q, want %q", got, "hello\n")
	}
}

func TestReadUntilIdleNoData(t *testing.T) {
	c := NewConn(newFakePort(), ModeCXR)
	if _, err := c.readUntilIdle(); err == nil {
		t.Fatal("expected error when port has no data")
	}
}

func TestCommandUnknownMode(t *testing.T) {
	c := &Conn{RW: newFakePort(), Mode: Mode(99)}
	if _, err := c.Command("x"); err == nil {
		t.Fatal("expected error for unknown mode")
	}
}

func TestParseMode(t *testing.T) {
	for in, want := range map[string]Mode{
		"cxr": ModeCXR, "CXR": ModeCXR,
		"cxrf": ModeCXRF,
		"sw": ModeSW, "SW": ModeSW,
	} {
		got, err := ParseMode(in)
		if err != nil || got != want {
			t.Errorf("ParseMode(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseMode("goku"); err == nil {
		t.Error("ParseMode(goku) should fail")
	}
}
