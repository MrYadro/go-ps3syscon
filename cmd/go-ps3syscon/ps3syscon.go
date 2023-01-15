package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/chzyer/readline"
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

var completer = readline.NewPrefixCompleter(
	readline.PcItem("quit"),
)

func fillPc(cmdList map[string]map[string]string) {
	for k, v := range cmdList {
		cmd := readline.PcItem(k)
		scmd := strings.Split(v["subcommands"], ",")
		for _, sc := range scmd {
			cmd.Children = append(cmd.Children, readline.PcItem(sc))
		}
		completer.Children = append(completer.Children, cmd)
	}
}

func main() {
	sc := newSyscon(portName, sysconMode, noVerify)
	switch sysconMode {
	case "cxrf":
		{
			fillPc(intCmd)
		}
	case "cxr", "sw":
		{
			fillPc(extCmd)
		}
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[31mps3syscon>\033[0m ",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()
	l.CaptureExitSignal()

	log.SetOutput(l.Stderr())
	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		switch {
		case line == "quit":
			goto exit
		case line == "":
		default:
			resp, err := sc.proccessCommand(line)
			if err != nil {
				log.Print(err)
			}
			fmt.Println(resp)
		}
	}
exit:
}
