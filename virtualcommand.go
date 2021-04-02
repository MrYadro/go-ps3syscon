package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"strings"
)

var (
	SC2TBKey, _      = hex.DecodeString("71F03F184C01C5EBC3F6A22A42BA9525") // https://www.psdevwiki.com/ps3/Keys
	TB2SCKey, _      = hex.DecodeString("907E730F4D4E0A0B7B75F030EB1D9D36")
	Auth1Response, _ = hex.DecodeString("3350BD7820345C29056A223BA220B323")

	Auth    = "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	Zero, _ = hex.DecodeString("00000000000000000000000000000000")

	Auth1ResponseHeader, _ = hex.DecodeString("10100000FFFFFFFF0000000000000000")
	Auth2RequestHeader, _  = hex.DecodeString("10010000000000000000000000000000")

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
		"3003": "POWER FAIL",
		"4":    "Data error",
	}
)

func IsVritualCommand(com string) bool {
	switch c := com; {
	case c == "auth",
		strings.HasPrefix(c, "errinfo"):
		return true
	}
	return false
}

func decode(ciphertext []byte) []byte {
	block, err := aes.NewCipher(SC2TBKey)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, Zero)

	mode.CryptBlocks(ciphertext, ciphertext)

	return ciphertext
}

func encode(plaintext []byte) []byte {
	if len(plaintext)%aes.BlockSize != 0 {
		panic("plaintext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(TB2SCKey)
	if err != nil {
		panic(err)
	}

	ciphertext := make([]byte, len(plaintext))

	mode := cipher.NewCBCEncrypter(block, Zero)
	mode.CryptBlocks(ciphertext, plaintext)

	return ciphertext
}

func (sc Syscon) VirtualCommandAuth() string {
	res := sc.ProccessCommand("scopen")
	if res == "SC_READY" {
		fmt.Printf("Successfully opened syscon\n%s\n", res)
		res = sc.ProccessCommand(Auth)
		if len(res) == 128 {
			fmt.Println("Right response length")
			resNew, _ := hex.DecodeString(res)
			if bytes.Equal(resNew[0:0x10], Auth1ResponseHeader) {
				fmt.Println("Right Auth1 response header")
				data := decode(resNew[0x10:0x40])
				if bytes.Equal(data[0x8:0x10], Zero[0x0:0x8]) && bytes.Equal(data[0x10:0x20], Auth1Response) && bytes.Equal(data[0x20:0x30], Zero) {
					fmt.Println("Right Auth1 response body")
					newData := append(data[0x8:0x10], data[0x0:0x8]...)
					newData = append(newData, Zero...)
					newData = append(newData, Zero...)
					Auth2Body := encode(newData)
					authBody := append(Auth2RequestHeader, Auth2Body...)
					h := fmt.Sprintf("%02X", authBody)
					res := sc.ProccessCommand(h)
					if strings.Contains(res, "SC_SUCCESS") {
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

// 0xa0093003
func ParseErrorCode(err string) string {
	stepNo := err[4:6]
	errCat := err[6:7]
	errNo := err[6:10]
	return fmt.Sprintf("%s on step %s with error info: %s", scErrors[errCat], stepNo, scErrors[errNo])
}

func (sc Syscon) VirtualCommandErrinfo(com string) string {
	errCode := strings.Split(com, " ")
	if len(errCode) < 2 {
		return "Please provide error code!"
	} else {
		return ParseErrorCode(errCode[1])
	}
}

func (sc Syscon) ProccessVirtualCommand(com string) string {
	switch c := com; {
	case c == "auth":
		return sc.VirtualCommandAuth()
	case strings.HasPrefix(c, "errinfo"):
		return sc.VirtualCommandErrinfo(com)
	}

	return com
}
