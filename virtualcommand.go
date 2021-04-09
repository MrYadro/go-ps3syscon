package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

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
)

func decode(ciphertext []byte) []byte {
	block, err := aes.NewCipher(sc2TBKey)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, zero)

	mode.CryptBlocks(ciphertext, ciphertext)

	return ciphertext
}

func encode(plaintext []byte) []byte {
	if len(plaintext)%aes.BlockSize != 0 {
		panic("plaintext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(tb2SCKey)
	if err != nil {
		panic(err)
	}

	ciphertext := make([]byte, len(plaintext))

	mode := cipher.NewCBCEncrypter(block, zero)
	mode.CryptBlocks(ciphertext, plaintext)

	return ciphertext
}

func (sc syscon) virtualCommandAuth() string {
	switch sc.mode {
	case "cxrf":
		res := sc.proccessCommand("scopen")
		if res == "SC_READY" {
			fmt.Printf("Successfully opened syscon\n%s\n", res)
			res = sc.proccessCommand(auth)
			if len(res) == 128 {
				fmt.Println("Right response length")
				resNew, _ := hex.DecodeString(res)
				if bytes.Equal(resNew[0:0x10], auth1ResponseHeader) {
					fmt.Println("Right Auth1 response header")
					data := decode(resNew[0x10:0x40])
					if bytes.Equal(data[0x8:0x10], zero[0x0:0x8]) && bytes.Equal(data[0x10:0x20], auth1Response) && bytes.Equal(data[0x20:0x30], zero) {
						fmt.Println("Right Auth1 response body")
						newData := append(data[0x8:0x10], data[0x0:0x8]...)
						newData = append(newData, zero...)
						newData = append(newData, zero...)
						auth2Body := encode(newData)
						authBody := append(auth2RequestHeader, auth2Body...)
						com := fmt.Sprintf("%02X", authBody)
						res := sc.proccessCommand(com)
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
	default:
		res := sc.proccessCommand("AUTH1 " + auth)
		if res[0] == 0 {
			resNew, _ := hex.DecodeString(res)
			if bytes.Equal(resNew[0:0x10], auth1ResponseHeader) {
				data := decode(resNew[0x10:0x40])
				if bytes.Equal(data[0x8:0x10], zero[0x0:0x8]) && bytes.Equal(data[0x10:0x20], auth1Response) && bytes.Equal(data[0x20:0x30], zero) {
					newData := append(data[0x8:0x10], data[0x0:0x8]...)
					newData = append(newData, zero...)
					newData = append(newData, zero...)
					auth2Body := encode(newData)
					authBody := append(auth2RequestHeader, auth2Body...)
					com := fmt.Sprintf("%02X", authBody)
					res := sc.proccessCommand(com)
					if res[0] == 0 {
						fmt.Println("Auth successful")
					}
				} else {
					fmt.Println("Wrong Auth1 response body")
				}
			} else {
				fmt.Println("Wrong Auth1 response header")
			}
		} else {
			return "Auth1 response invalid"
		}
	}
	return "Auth failed"
}

func parseErrorCode(err string) string {
	re := regexp.MustCompile(`0xa[A-Fa-f0-9]{3}[1-4][0-9][0-6f][0-9f]`)
	valErr := re.MatchString(err)
	if valErr {
		stepNo := err[4:6]
		errCat := err[6:7]
		errNo := err[6:10]
		return fmt.Sprintf("%s on step %s with error info: %s", scErrors[errCat], stepNo, scErrors[errNo])
	} else {
		return "Unknown error!"
	}
}

func (sc syscon) VirtualCommandErrinfo(cmd string) string {
	errCode := strings.Split(cmd, " ")
	if len(errCode) < 2 {
		return "Please provide error code!"
	} else {
		return parseErrorCode(errCode[1])
	}
}

func (sc syscon) proccessVirtualCommand(cmd string) string {
	switch c := cmd; {
	case c == "auth":
		return sc.virtualCommandAuth()
	case strings.HasPrefix(c, "errinfo"):
		return sc.VirtualCommandErrinfo(cmd)
	}
	return cmd
}
