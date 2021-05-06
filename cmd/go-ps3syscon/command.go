package main

import (
	"strings"
	"time"
)

func (sc syscon) getRespTime(cmd string) time.Duration {
	// Make this proper
	switch sc.mode {
	case "cxr", "sw":
		switch com := cmd; {
		case com == "bringup":
			return 12
		}
		return 5
	default:
		switch com := cmd; {
		case com == "bringup":
			return 3
		}
		return 1
	}
}

func isVritualCommand(cmd string) bool {
	switch c := cmd; {
	case c == "auth",
		strings.HasPrefix(c, "errinfo"):
		return true
	}
	return false
}

func (sc syscon) proccessCommand(cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	if isVritualCommand(cmd) {
		return sc.proccessVirtualCommand(cmd), nil // TODO: Error handling
	} else {
		sc.sendCommand(cmd)
		time.Sleep(sc.getRespTime(cmd) * time.Second)
		return sc.receiveCommand()
	}
}
