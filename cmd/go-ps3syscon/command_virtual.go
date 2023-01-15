package main

import (
	"strings"
)

func (sc syscon) proccessVirtualCommand(cmd string) string {
	switch c := cmd; {
	case c == "auth":
		return sc.virtualCommandAuth()
	case strings.HasPrefix(c, "errinfo"):
		return sc.VirtualCommandErrinfo(cmd)
	case strings.HasPrefix(c, "cmdinfo"):
		return sc.VirtualCommandCmdinfo(cmd)
	}
	return cmd
}
