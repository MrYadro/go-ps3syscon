package main

import (
	"fmt"
	"log"
	"strings"

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

func countChecksum(cmd string) byte {
	var sum byte
	cmdBytes := []byte(cmd)
	for _, v := range cmdBytes {
		sum += v
	}
	return sum
}

func (sc syscon) sendCXRCommand(cmd string) {
	checksum := countChecksum(cmd)
	fcmd := fmt.Sprintf("C:%02X:%s", checksum, cmd)
	maxSize := 15 // for some reason we need to split every 15 symbols
	length := len(fcmd)
	var j int
	for i := 0; i < length; i += maxSize {
		j += maxSize
		if j > length {
			// j = length
			break // And don'r send extra
		}
		fmt.Println(fcmd[i:j])
		sc.writeCommand(fcmd[i:j])
	}
	sc.writeCommand("\r\n")
}

func (sc syscon) sendCXRFCommand(cmd string) {
	sc.writeCommand(cmd + "\r\n")
}

func (sc syscon) sendSWCommand(cmd string) {
	// SendCmdLong ??????
	checksum := countChecksum(cmd)
	fcmd := fmt.Sprintf("C:%02X:%s\r\n", checksum, cmd)
	sc.writeCommand(fcmd)
}

func (sc syscon) receiveCXRCommand() string {
	// buff := make([]byte, 1000)
	// n, err := sc.port.Read(buff)
	// fmt.Printf("Read %v bytes\n", n)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(string(buff[:n]))

	// // test := strings.SplitAfterN(string(buff[:n]), "\n", 2)
	// reg, err := regexp.Compile("[^a-zA-Z0-9:]+")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// processedString := reg.ReplaceAllString(string(buff[:n]), "")
	// fmt.Println(processedString)
	// return processedString
	buff := make([]byte, 1000)
	for {
		n, err := sc.port.Read(buff)
		fmt.Printf("Read %v bytes\n", n)
		if err != nil {
			log.Fatal(err)
		}
		if n == 0 {
			fmt.Println("\nEOF")
			break
		}

		// if strings.Contains(string(buff[:n]), "\r\n") {
		// 	test := strings.SplitAfterN(string(buff[:n]), "\n", 2)
		fmt.Println(string(buff[:n]))
		// 	return strings.TrimSpace(test[1])
		// }
	}
	return ""
}

func (sc syscon) receiveCXRFCommand() string {
	buff := make([]byte, 1000)
	for {
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

func (sc syscon) receiveSWCommand() string {
	return "not impl"
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
