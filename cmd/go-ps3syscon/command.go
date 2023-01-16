package main

import (
	"strings"
	"time"
)

func (sc syscon) proccessCommand(cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	sc.sendCommand(cmd)
	time.Sleep(1 * time.Second)
	return sc.receiveCommand()
}
