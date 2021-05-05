package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

func (sc syscon) sendCXRFCommand(cmd string) {
	sc.writeCommand(cmd + "\r\n")
}

func (sc syscon) receiveCXRFCommand() (string, error) {
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
			resp := strings.SplitAfterN(string(buff[:n]), "\n", 2)
			return strings.TrimSpace(resp[1]), nil
		}
	}
	return "", errors.New("wrong response")
}
