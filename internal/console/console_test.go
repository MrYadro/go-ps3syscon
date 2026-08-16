package console

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/MrYadro/go-ps3syscon/internal/protocol"
)

type fakeCommander struct {
	mode    protocol.Mode
	lastCmd string
	result  protocol.Result
	err     error
	authOut string
	authErr error
}

func (f *fakeCommander) Command(cmd string) (protocol.Result, error) {
	f.lastCmd = cmd
	return f.result, f.err
}

func (f *fakeCommander) Auth() (string, error) { return f.authOut, f.authErr }

func testConsole(mode protocol.Mode, fc *fakeCommander) (*Console, *bytes.Buffer) {
	var log bytes.Buffer
	return New(fc, mode, &log), &log
}

func TestHandleLineErrinfo(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, quit := cons.HandleLine("errinfo 0xa0093003")
	if quit {
		t.Fatal("unexpected quit")
	}
	// README example: Fatal booting error on step 09 with error info: POWER FAIL
	if out != "Fatal booting error on step 09 with error info: POWER FAIL" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineErrinfoEmpty(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("errinfo")
	if out != "Please provide error code!" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineErrinfoUnknown(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("errinfo xyz")
	if out != "Unknown error!" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineCmdinfo(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("cmdinfo becount")
	want := "becount - Display bringup/shutdown count + Power-on time, command called with no parametres and no subcommands"
	if out != want {
		t.Fatalf("out = %q, want %q", out, want)
	}
}

func TestHandleLineCmdinfoExtTable(t *testing.T) {
	// cxr/sw use the extCmd table: FAN exists there but not in intCmd.
	cons, _ := testConsole(protocol.ModeCXR, &fakeCommander{})
	out, _ := cons.HandleLine("cmdinfo FAN")
	if strings.HasPrefix(out, "Wrong command") {
		t.Fatalf("FAN should resolve in extCmd table for cxr mode: %q", out)
	}
	cons2, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out2, _ := cons2.HandleLine("cmdinfo FAN")
	if out2 != "Wrong command" {
		t.Fatalf("FAN should not resolve in intCmd table for cxrf mode: %q", out2)
	}
}

func TestHandleLineCmdinfoWrong(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("cmdinfo nosuchcmd")
	if out != "Wrong command" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineAuth(t *testing.T) {
	fc := &fakeCommander{authOut: "Auth successful"}
	cons, _ := testConsole(protocol.ModeCXR, fc)
	out, _ := cons.HandleLine("auth")
	if out != "Auth successful" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineAuthError(t *testing.T) {
	fc := &fakeCommander{authErr: errors.New("boom")}
	cons, _ := testConsole(protocol.ModeCXR, fc)
	out, _ := cons.HandleLine("auth")
	if out != "Error: boom" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLinePassThrough(t *testing.T) {
	fc := &fakeCommander{result: protocol.Result{Code: 0, Lines: []string{"4.4.2"}}}
	cons, _ := testConsole(protocol.ModeCXR, fc)
	out, _ := cons.HandleLine("version")
	if fc.lastCmd != "version" {
		t.Fatalf("lastCmd = %q", fc.lastCmd)
	}
	if out != "00000000 4.4.2" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLinePassThroughError(t *testing.T) {
	fc := &fakeCommander{err: errors.New("cxr: wrong checksum")}
	cons, _ := testConsole(protocol.ModeCXR, fc)
	out, _ := cons.HandleLine("version")
	if out != "Error: cxr: wrong checksum" {
		t.Fatalf("out = %q", out)
	}
}

func TestHandleLineQuit(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXR, &fakeCommander{})
	_, quit := cons.HandleLine("quit")
	if !quit {
		t.Fatal("quit should set quit=true")
	}
}

func TestHandleLineHelp(t *testing.T) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	out, _ := cons.HandleLine("help")
	if !strings.HasPrefix(out, "commands:") {
		t.Fatalf("out = %q", out)
	}
	if !strings.Contains(out, "becount") { // intCmd entry for cxrf
		t.Fatalf("help missing intCmd entries")
	}
}

func TestFormatResult(t *testing.T) {
	if got := formatResult(protocol.ModeCXR, protocol.Result{Code: 0x12, Lines: []string{"A", "B"}}); got != "00000012 A B" {
		t.Fatalf("cxr: %q", got)
	}
	if got := formatResult(protocol.ModeSW, protocol.Result{Code: 0x12, Lines: []string{"A", "B"}}); got != "00000012\nA\nB" {
		t.Fatalf("sw: %q", got)
	}
	if got := formatResult(protocol.ModeSW, protocol.Result{Code: 0x12}); got != "00000012" {
		t.Fatalf("sw empty: %q", got)
	}
	if got := formatResult(protocol.ModeCXRF, protocol.Result{Raw: "SC_READY"}); got != "SC_READY" {
		t.Fatalf("cxrf: %q", got)
	}
}

func TestBatchWritesOutAndLog(t *testing.T) {
	fc := &fakeCommander{result: protocol.Result{Raw: "SCM-4.4.2"}}
	var out bytes.Buffer
	var log bytes.Buffer
	cons := New(fc, protocol.ModeCXRF, &log)
	cons.Out = &out
	cons.Batch([]string{"version", "quit", "auth"})
	if !strings.Contains(out.String(), "> version") || !strings.Contains(out.String(), "SCM-4.4.2") {
		t.Fatalf("batch out = %q", out.String())
	}
	if !strings.Contains(log.String(), "SCM-4.4.2") {
		t.Fatalf("log = %q", log.String())
	}
	if !strings.Contains(log.String(), "> version") {
		t.Fatalf("log must record the command, got %q", log.String())
	}
	if strings.Contains(out.String(), "> auth") {
		t.Fatal("quit must stop batch processing")
	}
}

// The Run (interactive REPL) path also logs accepted lines to LogFile, but
// it is untested-by-design here: Run requires a readline TTY. Batch above
// covers the shared "> cmd" log-line contract.
