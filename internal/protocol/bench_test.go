package protocol

import (
	"fmt"
	"testing"
)

// chunked splits b into size-byte pieces, as a slow serial port might.
func chunked(b []byte, size int) [][]byte {
	var chunks [][]byte
	for len(b) > size {
		chunks = append(chunks, b[:size])
		b = b[size:]
	}
	if len(b) > 0 {
		chunks = append(chunks, b)
	}
	return chunks
}

func BenchmarkReadUntilCRLFChunks(b *testing.B) {
	full := []byte("eepromcheck 5\r\nE2PROM CHECK : OK\r\n")
	chunks := chunked(full, 8)
	b.SetBytes(int64(len(full)))
	for b.Loop() {
		c := NewConn(newFakePort(chunks...), ModeCXRF)
		c.readUntilCRLF()
	}
}

func BenchmarkReadUntilIdleChunks(b *testing.B) {
	body := "OK 00000000"
	frame := []byte(fmt.Sprintf("R:%02X:%s\r\n", countChecksum(body), body))
	chunks := chunked(frame, 8)
	b.SetBytes(int64(len(frame)))
	for b.Loop() {
		c := NewConn(newFakePort(chunks...), ModeCXR)
		c.readUntilIdle()
	}
}

// Leftover garbage in the input buffer ahead of a valid frame — the
// case ResetInputBuffer exists to flush but does not always catch.
func BenchmarkReadUntilIdleNoise(b *testing.B) {
	body := "OK 00000000"
	frame := []byte(fmt.Sprintf("R:%02X:%s\r\n", countChecksum(body), body))
	chunks := chunked(frame, 8)
	noise := make([][]byte, 64)
	for i := range noise {
		noise[i] = []byte("........")
	}
	b.SetBytes(int64(64*8 + len(frame)))
	for b.Loop() {
		c := NewConn(newFakePort(append(noise, chunks...)...), ModeCXR)
		c.readUntilIdle()
	}
}
