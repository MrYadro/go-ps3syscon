package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/MrYadro/go-ps3syscon/internal/console"
	"github.com/MrYadro/go-ps3syscon/internal/protocol"
	"go.bug.st/serial"
)

type cmdList []string

func (l *cmdList) String() string { return strings.Join(*l, "; ") }

func (l *cmdList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

func main() {
	portName := flag.String("port", "", "serial port to use (see -list-ports)")
	modeStr := flag.String("mode", "cxrf", "syscon mode: cxr, cxrf or sw")
	listPorts := flag.Bool("list-ports", false, "list available serial ports and exit")
	logPath := flag.String("l", "", "append session output to this log file")
	verbose := flag.Bool("v", false, "verbose: print raw frames and byte counts")
	var exec cmdList
	flag.Var(&exec, "e", `command to run non-interactively; repeatable, e.g. -e "auth" -e "version"`)
	flag.Parse()

	if *listPorts {
		ports, err := serial.GetPortsList()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error listing ports:", err)
			os.Exit(1)
		}
		for _, p := range ports {
			fmt.Println(p)
		}
		return
	}

	mode, err := protocol.ParseMode(*modeStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *portName == "" {
		fmt.Fprintln(os.Stderr, "no port given: use -port (see -list-ports)")
		os.Exit(1)
	}

	var logWriter io.Writer
	if *logPath != "" {
		logFile, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening log file:", err)
			os.Exit(1)
		}
		defer logFile.Close()
		logWriter = logFile
	}

	rw, err := protocol.OpenSerial(*portName, mode, time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open serial port %s: %v\n", *portName, err)
		os.Exit(1)
	}
	defer rw.Close()

	conn := protocol.NewConn(rw, mode)
	conn.Verbose = *verbose

	cons := console.New(conn, mode, logWriter)
	if len(exec) > 0 {
		cons.Batch(exec)
		return
	}
	if err := cons.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
