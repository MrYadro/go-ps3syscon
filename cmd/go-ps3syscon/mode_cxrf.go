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
		"dve": {
			"subcommands": "help,set,save,show",
			"description": "DVE chip parameters",
		},
		"eepcsum": {
			"description": "Shows eeprom checksum",
		},
		"eepromcheck": {
			"description": "Check eeprom",
			"parametres":  "id",
		},
		"eeprominit": {
			"description": "Init eeprom",
			"parametres":  "id",
		},
		"ejectsw": {
			"description": "Eject switch",
		},
		"errlog": {
			"description": "Gets the error log",
		},
		"fancon": {
			"description": "Does nothing",
		},
		"fanconautotype": {
			"description": "Does nothing",
		},
		"fanconmode": {
			"subcommands": "get",
			"description": "Fan control mode",
		},
		"fanconpolicy": {
			"subcommands": "get,set,getini,setini",
			"description": "Fan control policy",
		},
		"fandiag": {
			"description": "Fan test",
		},
		"faninictrl": {
			"description": "Does nothing",
		},
		"fanpol": {
			"description": "Does nothing",
		},
		"fanservo": {
			"description": "Does nothing",
		},
		"fantbl": {
			"subcommands": "get,set,getini,setini,gettable,settable",
			"description": "Fan table",
		},
		"firmud": {
			"description": "Firmware update",
		},
		"geterrlog": {
			"description": "Gets error log",
			"parametres":  "id",
		},
		"getrtc": {
			"description": "Gets rtc",
		},
		"halt": {
			"description": "Halts syscon",
		},
		"hdmi": {
			"description": "HDMI (various commands, use help)",
		},
		"hdmiid": {
			"description": "Get HDMI id's",
		},
		"hdmiid2": {
			"description": "Get HDMI id's",
		},
		"hversion": {
			"description": "Platform ID",
		},
		"hyst": {
			"subcommands": "get,set,getini,setini",
			"description": "Temperature zones",
		},
		"lasterrlog": {
			"description": "Last error from log",
		},
		"ledmode": {
			"description": "Get led mode",
			"parametres":  "id,id",
		},
		"LS": {
			"description": "LabStation Mode",
		},
		"ltstest": {
			"subcommands": "get,set",
			"description": "?Temp related? values",
		},
		"osbo": {
			"description": "Sets 0x2000F60",
		},
		"patchcsum": {
			"description": "Patch checksum",
		},
		"patchvereep": {
			"description": "Patch version eeprom",
		},
		"patchverram": {
			"description": "Patch version ram",
		},
		"poll": {
			"description": "Poll log",
		},
		"portscan": {
			"description": "Scan port",
			"parametres":  "port",
		},
		"powbtnmode": {
			"description": "Power button mode",
			"parametres":  "mode",
		},
		"powerstate": {
			"description": "Get power state",
		},
		"powersw": {
			"description": "Power switch",
		},
		"powupcause": {
			"description": "Power up cause",
		},
		"printmode": {
			"description": "Set printmode",
			"parametres":  "mode",
		},
		"printpatch": {
			"description": "Prints patch",
		},
		"r": {
			"description": "Read byte from SC",
			"parametres":  "offset,length",
		},
		"r16": {
			"description": "Read word from SC",
			"parametres":  "offset,length",
		},
		"r32": {
			"description": "Read dword from SC",
			"parametres":  "offset,length",
		},
		"r64": {
			"description": "Read qword from SC",
			"parametres":  "offset,length",
		},
		"r64d": {
			"description": "Read qword data from SC",
			"parametres":  "offset,length",
		},
		"rbe": {
			"description": "Read from BE",
			"parametres":  "offset",
		},
		"recv": {
			"description": "Receive something",
		},
		"resetsw": {
			"description": "Reset switch",
		},
		"restartlogerrtoeep": {
			"description": "Reenable error logging to eeprom",
		},
		"revision": {
			"description": "Get softid",
		},
		"rrsxc": {
			"description": "Read from RSX",
			"parametres":  "offset,length",
		},
		"rtcreset": {
			"description": "Reset RTC",
		},
		"scagv2": {
			"description": "Auth related?",
		},
		"scasv2": {
			"description": "Auth related?",
		},
		"scclose": {
			"description": "Close syscon",
		},
		"scopen": {
			"description": "Open syscon",
		},
		"send": {
			"description": "Send something",
			"parametres":  "variable",
		},
		"shutdown": {
			"description": "PS3 shutdown",
		},
		"startlogerrtsk": {
			"description": "Start error log task",
		},
		"stoplogerrtoeep": {
			"description": "Stop error logging to eeprom",
		},
		"stoplogerrtsk": {
			"description": "Stop error log task",
		},
		"syspowdown": {
			"description": "System power down",
			"parametres":  "param,param,param",
		},
		"task": {
			"description": "Print tasks",
		},
		"thalttest": {
			"description": "Does nothing",
		},
		"thermfatalmode": {
			"subcommands": "canboot,cannotboot",
			"description": "Set thermal boot mode",
		},
		"therrclr": {
			"description": "Thermal register clear",
		},
		"thrm": {
			"description": "Does nothing",
		},
		"tmp": {
			"description": "Get temperature",
			"parametres":  "zone",
		},
		"trace": {
			"description": "Trace tasks (use help)",
		},
		"trp": {
			"subcommands": "get,set,getini,setini",
			"description": "Temperature zones",
		},
		"tsensor": {
			"description": "Get raw temperature",
			"parametres":  "sensor",
		},
		"tshutdown": {
			"subcommands": "get,set,getini,setini",
			"description": "Thermal shutdown",
		},
		"tshutdowntime": {
			"description": "Thermal shutdown time",
			"parametres":  "time",
		},
		"tzone": {
			"description": "Show thermal zones",
		},
		"version": {
			"description": "SC firmware version",
		},
		"w": {
			"description": "Write byte to SC",
			"parametres":  "offset,value",
		},
		"w16": {
			"description": "Write word to SC",
			"parametres":  "offset,value",
		},
		"w32": {
			"description": "Write bword to SC",
			"parametres":  "offset,value",
		},
		"w64": {
			"description": "Write qword to SC",
			"parametres":  "offset,value",
		},
		"wbe": {
			"description": "Write to BE",
			"parametres":  "offset,value",
		},
		"wmmto": {
			"subcommands": "get",
			"description": "Get watchdog timeout",
		},
		"wrsxc": {
			"description": "Write to RSX",
			"parametres":  "offset,value",
		},
		"xdrdiag": {
			"subcommands": "start,info,result",
			"description": "XDR diag",
		},
		"xiodiag": {
			"description": "XIO diag",
		},
		"xrcv": {
			"description": "Xmodem receive",
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
