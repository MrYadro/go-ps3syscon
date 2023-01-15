package main

func (sc syscon) commandAuth() string {
	switch sc.mode {
	case "cxrf":
		return sc.commandCXRFAuth()
	default:
		return sc.commandCXRAuth()
	}
}
