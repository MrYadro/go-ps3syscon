package main

import (
	"fmt"
	"log"

	"go.bug.st/serial"
)

func newSyscon(pName, sMode string) syscon {
	var mode *serial.Mode
	switch sMode {
	case "cxrf":
		{
			mode = &serial.Mode{
				BaudRate: 115200,
			}
		}
	case "cxr", "sw":
		{
			mode = &serial.Mode{
				BaudRate: 57600,
			}
			fmt.Println(mode)
		}
	}
	port, err := serial.Open(pName, mode)
	if err != nil {
		log.Fatal(err)
	}
	return syscon{port: port, mode: sMode}
}

func (sc syscon) sendCommand(cmd string) {
	switch sc.mode {
	case "cxr":
		sc.sendCXRCommand(cmd)
	case "cxrf":
		sc.sendCXRFCommand(cmd)
	case "sw":
		sc.sendSWCommand(cmd)
	}
}

func (sc syscon) writeCommand(cmd string) {
	n, err := sc.port.Write([]byte(cmd))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Sent %v bytes\n", n)
}

func (sc syscon) receiveCommand() string {
	switch sc.mode {
	case "cxr":
		return sc.receiveCXRCommand()
	case "cxrf":
		return sc.receiveCXRFCommand()
	case "sw":
		return sc.receiveSWCommand()
	}
	return "wrong mode"
}
