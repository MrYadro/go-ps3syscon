package main

import "fmt"

func (sc syscon) sendSWCommand(cmd string) {
	// SendCmdLong ??????
	checksum := countChecksum(cmd)
	fcmd := fmt.Sprintf("C:%02X:%s\r\n", checksum, cmd)
	sc.writeCommand(fcmd)
}

func (sc syscon) receiveSWCommand() string {
	return "not impl"
}
