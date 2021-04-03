package main

import (
	"strings"
	"time"
)

func getRespTime(cmd string) time.Duration {
	switch com := cmd; {
	case com == "bringup":
		return 3
	}
	return 1
}

func isVritualCommand(cmd string) bool {
	switch c := cmd; {
	case c == "auth",
		strings.HasPrefix(c, "errinfo"):
		return true
	}
	return false
}

func (sc syscon) proccessCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if isVritualCommand(cmd) {
		return sc.proccessVirtualCommand(cmd)
	} else {
		sc.sendCommand(cmd)
		time.Sleep(getRespTime(cmd) * time.Second)
		return sc.receiveCommand()
	}
}
