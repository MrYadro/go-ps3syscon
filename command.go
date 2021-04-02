package main

import (
	"strings"
	"time"
)

func GetRespTime(com string) time.Duration {
	switch cmd := com; {
	case cmd == "bringup":
		return 3
	}
	return 1
}

func (sc Syscon) ProccessCommand(com string) string {
	com = strings.TrimSpace(com)
	if IsVritualCommand(com) {
		return sc.ProccessVirtualCommand(com)
	} else {
		sc.SendCommand(com)
		time.Sleep(GetRespTime(com) * time.Second)
		return sc.ReceiveCommand()
	}
}
