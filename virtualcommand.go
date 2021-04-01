package main

import (
	"fmt"
	"strings"
)

var (
	scErrors = map[string]string{
		"1":    "System error",
		"1001": "BE VRAM Power Fail",
		"1002": "RSX VRAM Power Fail",
		"1004": "AC/DC Power Fail",
		"1103": "Thermal Error",
		"1200": "BE Thermal Error",
		"1201": "RSX Thermal Error",
		"1203": "BE VR Thermal Error",
		"1204": "SB Thermal Error",
		"1205": "EE+GS Thermal Error",
		"1301": "BE PLL Unlick",
		"14FF": "Check Stop",
		"1601": "BE Livelock Detection",
		"1701": "BE Attention",
		"1802": "RSX INIT",
		"1900": "RTC Error (voltage drop)",
		"1901": "RTC Error (oscillation stop)",
		"1902": "RTC Error (Access Error)",
		"2":    "Fatal error",
		"3":    "Fatal booting error",
		"3003": "POWER FAIL",
		"4":    "Data error",
	}
)

func IsVritualCommand(com string) bool {
	switch c := com; {
	case c == "auth",
		strings.HasPrefix(c, "errinfo"):
		return true
	}
	return false
}

func (sc Syscon) VirtualCommandAuth() string {
	return "AUTH"
}

// 0xa0093003
func ParseErrorCode(err string) string {
	stepNo := err[4:6]
	errCat := err[6:7]
	errNo := err[6:10]
	return fmt.Sprintf("%s on step %s with error info: %s", scErrors[errCat], stepNo, scErrors[errNo])
}

func (sc Syscon) VirtualCommandErrinfo(com string) string {
	errCode := strings.Split(com, " ")
	if len(errCode) < 2 {
		return "Please provide error code!"
	} else {
		return ParseErrorCode(errCode[1])
	}
}

func (sc Syscon) ProccessVirtualCommand(com string) string {
	switch c := com; {
	case c == "auth":
		return sc.VirtualCommandAuth()
	case strings.HasPrefix(c, "errinfo"):
		return sc.VirtualCommandErrinfo(com)
	}

	return com
}
