package main

import (
	"errors"
	"fmt"
	"log"

	"go.bug.st/serial"
)

var (
	cmdList map[string]map[string]string
)

func newSyscon(pName, sMode string) syscon {
	var pMode *serial.Mode
	switch sMode {
	case "cxrf":
		{
			pMode = &serial.Mode{
				BaudRate: 115200,
			}
			cmdList = intCmd
			fillPc(intCmd)
		}
	case "cxr", "sw":
		{
			pMode = &serial.Mode{
				BaudRate: 57600,
			}
			cmdList = extCmd
			fillPc(extCmd)
		}
	}
	port, err := serial.Open(pName, pMode)
	if err != nil {
		fmt.Printf("Could not find serial: %s\n", pName)
		log.Fatal(err)
	}
	port.ResetInputBuffer()
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

func (sc syscon) receiveCommand() (string, error) {
	switch sc.mode {
	case "cxr":
		return sc.receiveCXRCommand()
	case "cxrf":
		return sc.receiveCXRFCommand()
	case "sw":
		return sc.receiveSWCommand()
	}
	return "wrong mode", errors.New("wrong mode")
}
