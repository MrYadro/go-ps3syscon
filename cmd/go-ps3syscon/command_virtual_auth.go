package main

func (sc syscon) virtualCommandAuth() string {
	switch sc.mode {
	case "cxrf":
		return sc.virtualCommandCXRFAuth()
	default:
		return sc.virtualCommandCXRAuth()
	}
}
