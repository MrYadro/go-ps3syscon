package main

import (
	"fmt"
	"log"
	"strings"
)

func (sc syscon) sendCXRFCommand(cmd string) {
	sc.writeCommand(cmd + "\r\n")
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
