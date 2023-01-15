package main

import (
	"fmt"
	"strings"
)

func (sc syscon) VirtualCommandCmdinfo(cmd string) string {
	command := strings.Split(cmd, " ")
	cm, ok := cmdList[command[1]]
	if ok {
		params, ok := cm["parametres"]
		if ok {
			params = strings.Join(strings.Split(params, ","), ", ")
		} else {
			params = "no"
		}
		subcmd, ok := cm["subcommands"]
		if ok {
			subcmd = strings.Join(strings.Split(subcmd, ","), ", ")
		} else {
			subcmd = "no"
		}
		return fmt.Sprintf("%s - %s, command called with %s parametres and %s subcommands", command[1], cm["description"], params, subcmd)
	}
	return "Wrong command"
}
