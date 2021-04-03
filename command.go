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

func isVritualCommand(com string) bool {
	switch c := com; {
	case c == "auth",
		strings.HasPrefix(c, "errinfo"):
		return true
	}
	return false
}

func (sc syscon) proccessCommand(com string) string {
	com = strings.TrimSpace(com)
	if isVritualCommand(com) {
		return sc.proccessVirtualCommand(com)
	} else {
		sc.sendCommand(com)
		time.Sleep(getRespTime(com) * time.Second)
		return sc.receiveCommand()
	}
}
