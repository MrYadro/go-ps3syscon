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
)

type syscon struct {
	port serial.Port
	mode string
}

type test struct {
	mode string
}

func init() {
	flag.StringVar(&portName, "port", "/dev/tty.SLAB_USBtoUART", "port to use")
	flag.StringVar(&sysconMode, "mode", "CXRF", "syscon mode")
	flag.Parse()
	sysconMode = strings.ToLower(sysconMode)
}

func main() {
	sc := newSyscon(portName, sysconMode)

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		cmd, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		resp := sc.proccessCommand(cmd)
		fmt.Println(resp)
	}
}
