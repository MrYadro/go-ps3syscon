package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
)

func (sc syscon) sendCXRFCommand(cmd string) {
	sc.writeCommand(cmd + "\r\n")
}

func (sc syscon) receiveCXRFCommand() (string, error) {
	buff := make([]byte, 1500)
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

func (sc syscon) commandCXRFAuth() string {
	res, err := sc.proccessCommand("scopen")
	if err != nil || res != "SC_READY" {
		return fmt.Sprintf("Error opening syscon\n%s\n", res)
	}
	fmt.Printf("Successfully opened syscon\n%s\n", res)

	res, err = sc.proccessCommand(auth)
	if err != nil || len(res) != 128 {
		return "Wrong response length"
	}
	fmt.Println("Right response length")

	resHex, _ := hex.DecodeString(res)
	if !bytes.Equal(resHex[0:0x10], auth1ResponseHeader) {
		return "Wrong Auth1 response header"
	}
	fmt.Println("Right Auth1 response header")

	data := decode(resHex[0x10:0x40])
	if !bytes.Equal(data[0x8:0x10], zero[0x0:0x8]) || !bytes.Equal(data[0x10:0x20], auth1Response) || !bytes.Equal(data[0x20:0x30], zero) {
		return "Wrong Auth1 response body"
	}
	fmt.Println("Right Auth1 response body")

	newData := append(data[0x8:0x10], data[0x0:0x8]...)
	newData = append(newData, zero...)
	newData = append(newData, zero...)
	auth2Body := encode(newData)
	authBody := append(auth2RequestHeader, auth2Body...)
	com := fmt.Sprintf("%02X", authBody)
	res, err = sc.proccessCommand(com)
	if err != nil || !strings.Contains(res, "SC_SUCCESS") {
		return "Auth failed"
	}
	return "Auth successful"
}
