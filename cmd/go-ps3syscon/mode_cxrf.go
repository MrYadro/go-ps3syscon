package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
)

var (
	intCmd = map[string]map[string]string{
		"becount": {
			"description": "Display bringup/shutdown count + Power-on time",
		},
		"bepgoff": {
			"description": "BE power grid off",
		},
		"bepkt": {
			"subcommands": "show,set,unset,mode,debug,help",
			"description": "Packet permissions",
		},
		"bestat": {
			"description": "Get status of BE",
		},
		"boardconfig": {
			"description": "Displays board configuration",
		},
		"bootbeep": {
			"subcommands": "stat,on,off",
			"description": "Boot beep",
		},
		"bringup": {
			"description": "Turn PS3 on",
		},
		"bsn": {
			"description": "Get board serial number",
		},
		"bstatus": {
			"description": "HDMI related status",
		},
		"buzz": {
			"description": "Activate buzzer",
			"parametres":  "freq",
		},
		"buzzpattern": {
			"description": "Buzzer pattern",
			"parametres":  "freq,pattern,count",
		},
		"clear_err": {
			"subcommands": "last,eeprom,all",
			"description": "Clear errors",
		},
		"clearerrlog": {
			"description": "Clears error log",
		},
		"comm": {
			"description": "Communication mode",
		},
		"commt": {
			"subcommands": "help,start,stop,send",
			"description": "Manual BE communication",
		},
		"cp": {
			"subcommands": "ready,busy,reset,beepremote,beep2kn1n3,beep2kn2n3",
			"description": "CP control commands",
		},
		"csum": {
			"description": "Firmware checksum",
		},
		"devpm": {
			"subcommands": "ata,pci,pciex,rsx",
			"description": "Device power management",
		},
		"diag": {
			"description": "Diag (execute without paramto show help)",
		},
		"disp_err": {
			"description": "Displays errors",
		},
		"duty": {
			"subcommands": "get,set,getmin,setmin,getmax,setmax,getinmin,setinmin,getinmax,setinmax",
			"description": "Fan policy",
		},
	}
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

func (sc syscon) virtualCommandCXRFAuth() string {
	res, err := sc.proccessCommand("scopen")
	if err == nil && res == "SC_READY" {
		fmt.Printf("Successfully opened syscon\n%s\n", res)
		res, err = sc.proccessCommand(auth)
		if err == nil && len(res) == 128 {
			fmt.Println("Right response length")
			resHex, _ := hex.DecodeString(res)
			if bytes.Equal(resHex[0:0x10], auth1ResponseHeader) {
				fmt.Println("Right Auth1 response header")
				data := decode(resHex[0x10:0x40])
				if bytes.Equal(data[0x8:0x10], zero[0x0:0x8]) && bytes.Equal(data[0x10:0x20], auth1Response) && bytes.Equal(data[0x20:0x30], zero) {
					fmt.Println("Right Auth1 response body")
					newData := append(data[0x8:0x10], data[0x0:0x8]...)
					newData = append(newData, zero...)
					newData = append(newData, zero...)
					auth2Body := encode(newData)
					authBody := append(auth2RequestHeader, auth2Body...)
					com := fmt.Sprintf("%02X", authBody)
					res, err := sc.proccessCommand(com)
					if err == nil && strings.Contains(res, "SC_SUCCESS") {
						return "Auth successful"
					}
				} else {
					fmt.Println("Wrong Auth1 response body")
				}
			} else {
				fmt.Println("Wrong Auth1 response header")
			}
		} else {
			fmt.Println("Wrong response length")
		}
	} else {
		fmt.Println("Error opening syscon")
	}
	return "Auth failed"
}
