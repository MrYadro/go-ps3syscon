package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

func (sc syscon) sendCXRCommand(cmd string) {
	checksum := countChecksum(cmd)
	fcmd := fmt.Sprintf("C:%02X:%s", checksum, cmd)
	maxSize := 15 // for some reason we need to split every 15 symbols
	length := len(fcmd)
	var j int
	for i := 0; i < length; i += maxSize {
		j += maxSize
		if j > length {
			j = length
		}
		fmt.Println(fcmd[i:j])
		sc.writeCommand(fcmd[i:j])
	}
	sc.writeCommand("\r\n")
}

func (sc syscon) receiveCXRCommand() (string, error) {
	buff := make([]byte, 1000)
	n, err := sc.port.Read(buff)
	fmt.Printf("Read %v bytes\n", n)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buff[:n]))
	respRaw := strings.TrimSpace(string(buff[:n]))
	resp := strings.Split(respRaw, ":")
	if len(resp) != 3 {
		fmt.Println("wrong response length")
		return "", errors.New("wrong response length")
	}
	if resp[0] != "R" && resp[0] != "E" {
		fmt.Println("magic?")
		return "", errors.New("magic?")
	}
	if resp[1] != fmt.Sprintf("%02X", countChecksum(resp[2])) {
		fmt.Println("wrong checksum")
		return "", errors.New("wrong chechsum")
	}
	respData := strings.Split(resp[2], " ")
	if resp[0] == "R" && len(respData) < 2 || resp[0] == "E" && len(respData) != 2 {
		fmt.Println("wrong data length")
		return "", errors.New("wrong data length")
	}
	if respData[0] != "OK" || len(respData) < 2 {
		respCode, err := strconv.Atoi(respData[1])
		if err == nil && respCode == 0 {
			return respData[2], nil
		}
		return "", errors.New("wrong response code")
	}
	respCode, err := strconv.Atoi(respData[1])
	if err == nil && respCode == 0 {
		return respData[2], nil
	}
	return "", errors.New("wrong response code")
}

func (sc syscon) commandCXRAuth() string {
	res, err := sc.proccessCommand("AUTH1 " + auth)
	resHex, _ := hex.DecodeString(res)
	if err != nil || !bytes.Equal(resHex[0:0x10], auth1ResponseHeader) {
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
	com := fmt.Sprintf("AUTH2 %02X", authBody)
	_, err = sc.proccessCommand(com)
	if err != nil {
		return "Auth failed"
	}
	return "Auth successful"
}
