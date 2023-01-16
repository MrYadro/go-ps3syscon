package main

import (
	"strings"
	"time"
)

func (sc syscon) getRespTime(cmd string) time.Duration {
	//TODO: Make this proper
	switch sc.mode {
	case "cxr", "sw":
		return 2
	default:
		switch com := cmd; {
		case com == "bringup":
			return 3
		}
		return 1
	}
}

func (sc syscon) proccessCommand(cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	sc.sendCommand(cmd)
	time.Sleep(sc.getRespTime(cmd) * time.Second)
	return sc.receiveCommand()
}
