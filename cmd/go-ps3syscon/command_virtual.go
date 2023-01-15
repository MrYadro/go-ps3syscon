package main

func (sc syscon) proccessVirtualCommand(cmd string) string {
	switch c := cmd; {
	case c == "auth":
		return sc.virtualCommandAuth()
	}
	return cmd
}
