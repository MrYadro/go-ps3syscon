package protocol

import (
	"io"
	"sync"
)

// fakePort is a scripted io.ReadWriteCloser: each queued []byte is served by
// one Read call; once exhausted, Read reports idle with (0, errIdle).
type fakePort struct {
	mu        sync.Mutex
	responses [][]byte
	readIdx   int
	Written   []byte
	Writes    [][]byte
	Closed    bool
}

var errIdle = &idleError{}

type idleError struct{}

func (*idleError) Error() string { return "fake port idle" }

func newFakePort(reads ...[]byte) *fakePort {
	return &fakePort{responses: reads}
}

func (f *fakePort) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readIdx >= len(f.responses) {
		return 0, errIdle
	}
	n := copy(p, f.responses[f.readIdx])
	f.readIdx++
	return n, nil
}

func (f *fakePort) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Writes = append(f.Writes, append([]byte{}, p...))
	f.Written = append(f.Written, p...)
	return len(p), nil
}

func (f *fakePort) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Closed = true
	return nil
}

var _ io.ReadWriteCloser = (*fakePort)(nil)
