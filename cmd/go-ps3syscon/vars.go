package main

import "encoding/hex"

var (
	sc2TBKey, _      = hex.DecodeString("71F03F184C01C5EBC3F6A22A42BA9525") // https://www.psdevwiki.com/ps3/Keys
	tb2SCKey, _      = hex.DecodeString("907E730F4D4E0A0B7B75F030EB1D9D36")
	auth1Response, _ = hex.DecodeString("3350BD7820345C29056A223BA220B323")

	auth    = "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	zero, _ = hex.DecodeString("00000000000000000000000000000000")

	auth1ResponseHeader, _ = hex.DecodeString("10100000FFFFFFFF0000000000000000")
	auth2RequestHeader, _  = hex.DecodeString("10010000000000000000000000000000")

	scErrors = map[string]string{
		"1":    "System error",
		"1001": "BE VRAM Power Fail",
		"1002": "RSX VRAM Power Fail",
		"1004": "AC/DC Power Fail",
		"1103": "Thermal Error",
		"1200": "BE Thermal Error",
		"1201": "RSX Thermal Error",
		"1203": "BE VR Thermal Error",
		"1204": "SB Thermal Error",
		"1205": "EE+GS Thermal Error",
		"1301": "BE PLL Unlick",
		"14FF": "Check Stop",
		"1601": "BE Livelock Detection",
		"1701": "BE Attention",
		"1802": "RSX INIT",
		"1900": "RTC Error (voltage drop)",
		"1901": "RTC Error (oscillation stop)",
		"1902": "RTC Error (Access Error)",
		"2":    "Fatal error",
		"2001": "BE Error (IC1001)",
		"2002": "RSX Error (IC2001)",
		"2003": "SB Error (IC3001)",
		"2010": "Clock Generator Error (IC5001)",
		"2011": "Clock Generator Error (IC5003)",
		"2012": "Clock Generator Error (IC5002)",
		"2013": "Clock Generator Error (IC5004)",
		"2020": "HDMI Error (IC2502)",
		"2022": "DVE Error (IC2406)",
		"2030": "Thermal Sensor Error (IC1101)",
		"2031": "Thermal Sensor Error (IC2101)",
		"2033": "Thermal Sensor Error (IC3101)",
		"2101": "BE Error (IC1001)",
		"2102": "RSX Error (IC2001)",
		"2103": "SB Error (IC3001",
		"2110": "Clock Generator Error (IC5001)",
		"2111": "Clock Generator Error (IC5003)",
		"2112": "Clock Generator Error (IC5002)",
		"2113": "Clock Generator Error (IC5004)",
		"2120": "HDMI Error (IC2502)",
		"2122": "DVE Error (IC2406)",
		"2130": "Thermal Sensor Error (IC1101)",
		"2131": "Thermal Sensor Error (IC2101)",
		"2133": "Thermal Sensor Error (IC3101)",
		"2203": "SB Error (IC3001)",
		"3":    "Fatal booting error",
		"3000": "POWER FAIL",
		"3001": "POWER FAIL",
		"3002": "POWER FAIL",
		"3003": "POWER FAIL",
		"3004": "POWER FAIL",
		"3010": "BE Error (IC1001)",
		"3011": "BE Error (IC1001)",
		"3012": "BE Error (IC1001)",
		"3020": "BE Error (IC1001)",
		"3030": "BE Error (IC1001)",
		"3031": "BE Error (IC1001)",
		"3032": "BE Error (IC1001)",
		"3033": "BE Error (IC1001)",
		"3034": "BE Error (IC1001)",
		"3035": "BE-RSX Error (IC1001-IC2001)",
		"3036": "BE-RSX Error (IC1001-IC2001)",
		"3037": "BE-RSX Error (IC1001-IC2001)",
		"3038": "BE-SB Error (IC1001-IC3001)",
		"3039": "BE-SB Error (IC1001-IC3001)",
		"3040": "Flash controller Error (IC3801)",
		"4":    "Data error",
		"4001": "BE Error (IC1001)",
		"4002": "RSX Error (IC2001)",
		"4003": "SB Error (IC3001)",
		"4011": "BE Error (IC1001)",
		"4101": "BE Error (IC1001)",
		"4102": "RSX Error (IC2001)",
		"4103": "SB Error (IC3001)",
		"4111": "BE Error (IC1001)",
		"4201": "BE Error (IC1001)",
		"4202": "RSX Error (IC2001)",
		"4203": "SB Error (IC3001)",
		"4211": "BE Error (IC1001)",
		"4212": "RSX Error (IC2001)",
		"4221": "BE Error (IC1001)",
		"4222": "RSX Error (IC2001)",
		"4231": "BE Error (IC1001)",
		"4261": "BE Error (IC1001)",
		"4301": "BE Error (IC1001)",
		"4302": "RSX Error (IC2001)",
		"4303": "SB Error (IC3001)",
		"4311": "BE Error (IC1001)",
		"4312": "RSX Error (IC2001)",
		"4321": "BE Error (IC1001)",
		"4322": "RSX Error (IC2001)",
		"4332": "RSX Error (IC2001)",
		"4341": "BE Error (IC1001)",
		"4401": "BE or RSX Error (IC1001 or IC2001)",
		"4402": "BE or RSX Error (IC1001 or IC2001)",
		"4403": "BE or SB Error (IC1001 or IC3001)",
		"4411": "BE or RSX Error (IC1001 or IC2001)",
		"4412": "BE or RSX Error (IC1001 or IC2001)",
		"4421": "BE or RSX Error (IC1001 or IC2001)",
		"4422": "BE or RSX Error (IC1001 or IC2001)",
		"4432": "BE or RSX Error (IC1001 or IC2001)",
		"4441": "BE or SB Error (IC1001 or IC3001)",
	}

	extCmd = map[string]map[string]string{
		"BOOT": {
			"subcommands": "TEST,CONT",
			"description": "",
		},
		"SHUTDOWN":   {},
		"HALT":       {},
		"BOOTENABLE": {},
		"AUTH1":      {},
		"AUTH2":      {},
		"AUTHVER": {
			"subcommands": "SET,GET",
		},
		"EEP": {
			"subcommands": "INIT,SET,GET",
		},
		"PDAREA": {
			"subcommands": "SET,GET",
		},
		"CSAREA": {
			"subcommands": "SET,GET",
		},
		"VID": {
			"subcommands": "GET",
		},
		"CID": {
			"subcommands": "GET",
		},
		"ECID": {
			"subcommands": "GET",
		},
		"REV": {
			"subcommands": "SB",
		},
		"SPU": {
			"subcommands": "INFO",
		},
		"KSV": {},
		"FAN": {
			"subcommands": "SETPOLICY,GETPOLICY,START,STOP,SETDUTY,GETDUTY",
		},
		"R8":       {},
		"W8":       {},
		"R16":      {},
		"W16":      {},
		"R32":      {},
		"W32":      {},
		"RBE":      {},
		"WBE":      {},
		"PORTSTAT": {},
		"VER":      {},
		"BUZ":      {},
		"SERVFAN":  {},
		"ERRLOG": {
			"subcommands": "START,STOP,GET,CLEAR",
		},
	}

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
