package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"go.bug.st/serial"
)

var (
	portName   string
	sysconMode string
	noVerify   bool
)

type syscon struct {
	port     serial.Port
	mode     string
	noVerify bool
}

func init() {
	flag.StringVar(&portName, "port", "/dev/tty.usbserial-145410", "port to use")
	flag.StringVar(&sysconMode, "mode", "CXRF", "syscon mode")
	flag.BoolVar(&noVerify, "noverify", false, "verify command")
	flag.Parse()
	sysconMode = strings.ToLower(sysconMode)
}

func main() {
	sc := newSyscon(portName, sysconMode, noVerify)

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		cmd, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		resp, err := sc.proccessCommand(cmd)
		if err != nil {
			log.Print(err)
		}
		fmt.Println(resp)
	}
}
