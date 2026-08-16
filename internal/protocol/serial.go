package protocol

import (
	"io"
	"time"

	"go.bug.st/serial"
)

func baudFor(mode Mode) int {
	if mode == ModeCXRF {
		return 115200
	}
	return 57600
}

// OpenSerial opens the named port at the mode's baud rate with a read
// timeout, ready to be wrapped in a Conn. This is the only file in the
// package allowed to import the serial library.
func OpenSerial(portName string, mode Mode, timeout time.Duration) (io.ReadWriteCloser, error) {
	p, err := serial.Open(portName, &serial.Mode{BaudRate: baudFor(mode)})
	if err != nil {
		return nil, err
	}
	if err := p.SetReadTimeout(timeout); err != nil {
		p.Close()
		return nil, err
	}
	p.ResetInputBuffer()
	return p, nil
}
