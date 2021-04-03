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
	PortName   string
	SysconMode string
)

type syscon struct {
	port serial.Port
	mode string
}

func init() {
	flag.StringVar(&PortName, "port", "/dev/tty.SLAB_USBtoUART", "port to use")
	flag.StringVar(&SysconMode, "mode", "CXRF", "syscon mode")
	flag.Parse()
	SysconMode = strings.ToLower(SysconMode)
}

func main() {
	sc := newSyscon(PortName, SysconMode)

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
