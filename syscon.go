package main

import (
	"fmt"
	"log"
	"strings"

	"go.bug.st/serial"
)

func NewSyscon(pName, sMode string) Syscon {
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
		}
	}
	port, err := serial.Open(pName, mode)
	if err != nil {
		log.Fatal(err)
	}
	return Syscon{port: port, mode: sMode}
}

func (sc Syscon) SendCommand(com string) {
	switch sc.mode {
	case "cxr":
		fmt.Println("Not implemened")
	case "cxrf":
		n, err := sc.port.Write([]byte(com + "\r\n"))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Sent %v bytes\n", n)
	case "sw":
		fmt.Println("Not implemened")
	}
}

func (sc Syscon) ReceiveCommand() string {
	buff := make([]byte, 1000)
	for {
		// Reads up to 100 bytes
		n, err := sc.port.Read(buff)
		fmt.Printf("Read %v bytes\n", n)
		if err != nil {
			log.Fatal(err)
		}
		if n == 0 {
			fmt.Println("\nEOF")
			break
		}

		if strings.Contains(string(buff[:n]), "\r\n") {
			test := strings.SplitAfterN(string(buff[:n]), "\n", 2)
			return strings.TrimSpace(test[1])
		}
	}
	return ""
}
