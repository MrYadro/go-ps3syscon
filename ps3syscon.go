package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.bug.st/serial"
)

const (
	SC2TBKey      = "71F03F184C01C5EBC3F6A22A42BA9525" // https://www.psdevwiki.com/ps3/Keys
	TB2SCKey      = "907E730F4D4E0A0B7B75F030EB1D9D36"
	Auth1Response = "3350BD7820345C29056A223BA220B323"
	Auth2Response = "3C4689E97EDF5A86C6F174888D6085CF"
	Auth          = "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	Zero          = "00000000000000000000000000000000"

	Auth1ResponseHeader = "10100000FFFFFFFF0000000000000000"
	Auth2RequestHeader  = "10010000000000000000000000000000"
)

var (
	PortName   string
	SysconMode string
)

type Syscon struct {
	port serial.Port
	mode string
}

func init() {
	flag.StringVar(&PortName, "port", "/dev/tty.SLAB_USBtoUART", "port to use")
	flag.StringVar(&SysconMode, "mode", "CXRF", "syscon mode")
	flag.Parse()
	SysconMode = strings.ToLower(SysconMode)
}

func send(port serial.Port, command string) {
	n, err := port.Write([]byte(command + "\r\n"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Sent %v bytes\n", n)
}

func receive(port serial.Port) string {
	buff := make([]byte, 1000)
	for {
		// Reads up to 100 bytes
		n, err := port.Read(buff)
		fmt.Printf("Read %v bytes\n", n)
		if err != nil {
			log.Fatal(err)
		}
		if n == 0 {
			fmt.Println("\nEOF")
			break
		}

		// fmt.Printf("%s", string(buff[:n]))
		// fmt.Printf("%s", string(buff))

		// If we receive a newline stop reading
		if strings.Contains(string(buff[:n]), "\r\n") {
			// fmt.Println(string(buff[:n]))
			test := strings.SplitAfterN(string(buff[:n]), "\n", 2)
			// fmt.Printf("READ: %v\n", strings.TrimSpace(test[1]))
			return strings.TrimSpace(test[1])
		}
	}
	return ""
}

func command(port serial.Port, command string) string {
	send(port, command)
	time.Sleep(1 * time.Second)
	resp := receive(port)
	return resp
}

func decode(toDecode []byte) []byte {
	key, _ := hex.DecodeString(SC2TBKey)
	ciphertext := toDecode

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv, _ := hex.DecodeString(Zero)

	if len(ciphertext)%aes.BlockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	mode.CryptBlocks(ciphertext, ciphertext)

	return ciphertext
}

func encode(toEncode []byte) []byte {
	// Load your secret key from a safe place and reuse it across multiple
	// NewCipher calls. (Obviously don't use this example key for anything
	// real.) If you want to convert a passphrase to a key, use a suitable
	// package like bcrypt or scrypt.
	key, _ := hex.DecodeString(TB2SCKey)
	plaintext := toEncode

	// CBC mode works on blocks so plaintexts may need to be padded to the
	// next whole block. For an example of such padding, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. Here we'll
	// assume that the plaintext is already of the correct length.
	if len(plaintext)%aes.BlockSize != 0 {
		panic("plaintext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, len(plaintext))
	iv, _ := hex.DecodeString(Zero)
	// if _, err := io.ReadFull(rand.Reader, iv); err != nil {
	// 	panic(err)
	// }

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	// It's important to remember that ciphertexts must be authenticated
	// (i.e. by using crypto/hmac) as well as being encrypted in order to
	// be secure.

	// fmt.Printf("%x\n", ciphertext)
	return ciphertext
}

func auth(port serial.Port) {
	res := command(port, "scopen")
	if res == "SC_READY" {
		fmt.Printf("Successfully opened syscon\n%s\n", res)
		res = command(port, Auth)
		if len(res) == 128 {
			fmt.Println("Right response length")
			resNew, _ := hex.DecodeString(res)
			Auth1ResponseHeaderBin, _ := hex.DecodeString(Auth1ResponseHeader)
			if bytes.Equal(resNew[0:0x10], Auth1ResponseHeaderBin) {
				fmt.Println("Right Auth1 response header")
				// fmt.Println(res)
				// fmt.Println(res[0:0x10])
				// fmt.Println(res[0x10:0x40])
				data := decode(resNew[0x10:0x40])
				zerobin, _ := hex.DecodeString(Zero)
				Auth1ResponseBin, _ := hex.DecodeString(Auth1Response)
				if bytes.Equal(data[0x8:0x10], zerobin[0x0:0x8]) && bytes.Equal(data[0x10:0x20], Auth1ResponseBin) && bytes.Equal(data[0x20:0x30], zerobin) {
					fmt.Println("Right Auth1 response body")
					newData := append(data[0x8:0x10], data[0x0:0x8]...)
					newData = append(newData, zerobin...)
					newData = append(newData, zerobin...)
					Auth2Body := encode(newData)
					// fmt.Printf("AUTH2 %v\n", Auth2Body)
					Auth2RequestHeaderBin, _ := hex.DecodeString(Auth2RequestHeader)
					// fmt.Printf("AUTH2 HEADER%v\n", Auth2RequestHeaderBin)
					authBody := append(Auth2RequestHeaderBin, Auth2Body...)
					h := fmt.Sprintf("%02X", authBody)
					// fmt.Printf("AUTH2BODY %v\n", h)
					res := command(port, h)
					// fmt.Println(res)
					if strings.Contains(res, "SC_SUCCESS") {
						fmt.Println("Auth successful")
						res := command(port, "becount")
						fmt.Printf("%v", res)
					} else {
						fmt.Println("Auth failed")
					}
				} else {
					fmt.Println("Wrong Auth1 response body")
				}
			} else {
				fmt.Println("Wrong Auth1 response header")
			}
			// fmt.Printf("RES % x\n", res)
			// fmt.Printf("RES string %s\n", res)
			// fmt.Printf("RES length %d\n", len(res))
		} else {
			fmt.Println("Wrong response length")
		}
	} else {
		fmt.Println("Error opening syscon")
	}
}

func main() {
	sc := NewSyscon(PortName, SysconMode)

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		com, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		resp := sc.ProccessCommand(com)
		fmt.Println(resp)
	}
}
