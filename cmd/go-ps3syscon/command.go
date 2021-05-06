package main

import (
	"errors"
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

func isVritualCommand(cmd string) bool {
	switch c := cmd; {
	case c == "auth",
		strings.HasPrefix(c, "errinfo"), strings.HasPrefix(c, "cmdinfo"):
		return true
	}
	return false
}

func (sc syscon) valCmd(cmd string) error {
	if sc.noVerify {
		return nil
	}
	command := strings.Split(cmd, " ")

	cm, ok := cmdList[command[0]]
	if ok {
		if cm["subcommands"] != "" {
			subcommands := strings.Split(cm["subcommands"], ",")
			if len(subcommands) > 0 && len(command) > 1 {
				for _, subcommand := range subcommands {
					if command[1] == subcommand {
						return nil
					}
				}
				return errors.New("wrong subcommand")
			} else {
				return errors.New("subcommand missing")
			}
		}
	} else {
		return errors.New("wrong command")
	}
	return nil
}

func (sc syscon) proccessCommand(cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	if isVritualCommand(cmd) {
		return sc.proccessVirtualCommand(cmd), nil // TODO: Error handling
	} else {
		if sc.mode != "cxrf" {
			cmd = strings.ToUpper(cmd)
		}
		err := sc.valCmd(cmd)
		if err == nil {
			sc.sendCommand(cmd)
			time.Sleep(sc.getRespTime(cmd) * time.Second)
			return sc.receiveCommand()
		}
		return "", err
	}
}
