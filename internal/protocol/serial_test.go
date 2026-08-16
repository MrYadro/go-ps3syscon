package protocol

import "testing"

func TestBaudFor(t *testing.T) {
	for mode, want := range map[Mode]int{
		ModeCXR:  57600,
		ModeSW:   57600,
		ModeCXRF: 115200,
	} {
		if got := baudFor(mode); got != want {
			t.Errorf("baudFor(%v) = %d, want %d", mode, got, want)
		}
	}
}
