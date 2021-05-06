package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"regexp"
	"strings"
)

var ()

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
		return sc.virtualCommandCXRFAuth()
	default:
		return sc.virtualCommandCXRAuth()
	}
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

func (sc syscon) VirtualCommandCmdinfo(cmd string) string {
	command := strings.Split(cmd, " ")
	cm, ok := cmdList[command[1]]
	if ok {
		params, ok := cm["parametres"]
		if ok {
			params = strings.Join(strings.Split(params, ","), ", ")
		} else {
			params = "no"
		}
		subcmd, ok := cm["subcommands"]
		if ok {
			subcmd = strings.Join(strings.Split(subcmd, ","), ", ")
		} else {
			subcmd = "no"
		}
		return fmt.Sprintf("%s - %s, command called with %s parametres and %s subcommands", command[1], cm["description"], params, subcmd)
	}
	return "Wrong command"
}

func (sc syscon) proccessVirtualCommand(cmd string) string {
	switch c := cmd; {
	case c == "auth":
		return sc.virtualCommandAuth()
	case strings.HasPrefix(c, "errinfo"):
		return sc.VirtualCommandErrinfo(cmd)
	case strings.HasPrefix(c, "cmdinfo"):
		return sc.VirtualCommandCmdinfo(cmd)
	}
	return cmd
}
