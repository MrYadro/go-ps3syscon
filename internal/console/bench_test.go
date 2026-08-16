package console

import (
	"testing"

	"github.com/MrYadro/go-ps3syscon/internal/protocol"
)

func BenchmarkHelp(b *testing.B) {
	cons, _ := testConsole(protocol.ModeCXRF, &fakeCommander{})
	for b.Loop() {
		cons.HandleLine("help")
	}
}
