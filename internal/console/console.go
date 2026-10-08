// Package console implements the interactive syscon REPL and its
// built-in commands on top of the protocol package.
package console

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MrYadro/go-ps3syscon/internal/protocol"
	"github.com/chzyer/readline"
)

// Commander is the subset of protocol.Conn the console needs. Tests fake it.
type Commander interface {
	Command(cmd string) (protocol.Result, error)
	Auth() (string, error)
}

// Console is an interactive syscon session.
type Console struct {
	SC      Commander
	Mode    protocol.Mode
	LogFile io.Writer // optional: results are appended here
	Out     io.Writer // REPL/batch output; defaults to os.Stdout

	completer *readline.PrefixCompleter // built once for the mode
}

// New creates a Console. logFile may be nil.
func New(sc Commander, mode protocol.Mode, logFile io.Writer) *Console {
	return &Console{SC: sc, Mode: mode, LogFile: logFile, Out: os.Stdout, completer: newCompleter(mode)}
}

// HandleLine processes one input line, returning its output and whether
// the session should stop.
func (c *Console) HandleLine(line string) (string, bool) {
	line = strings.TrimSpace(line)
	switch {
	case line == "":
		return "", false
	case line == "quit":
		return "", true
	case line == "help":
		return c.usage(), false
	case line == "auth":
		out, err := c.SC.Auth()
		if err != nil {
			return "Error: " + err.Error(), false
		}
		return out, false
	case strings.HasPrefix(line, "errinfo"):
		return c.errinfo(strings.TrimSpace(strings.TrimPrefix(line, "errinfo"))), false
	case strings.HasPrefix(line, "cmdinfo"):
		return c.cmdinfo(strings.TrimSpace(strings.TrimPrefix(line, "cmdinfo"))), false
	default:
		res, err := c.SC.Command(line)
		if err != nil {
			return "Error: " + err.Error(), false
		}
		return formatResult(c.Mode, res), false
	}
}

// Batch runs commands non-interactively, stopping at quit.
func (c *Console) Batch(cmds []string) {
	for _, cmd := range cmds {
		fmt.Fprintf(c.Out, "> %s\n", cmd)
		c.logLine(cmd)
		out, quit := c.HandleLine(cmd)
		if out != "" {
			c.emit(out)
		}
		if quit {
			return
		}
	}
}

// Run starts the interactive REPL.
func (c *Console) Run() error {
	l, err := readline.NewEx(&readline.Config{
		Prompt:            "\033[31mps3syscon>\033[0m ",
		AutoComplete:      c.completer,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return err
	}
	defer l.Close()
	l.CaptureExitSignal()
	for {
		line, err := l.Readline()
		if err != nil { // ErrInterrupt or io.EOF (exit prompt)
			return nil
		}
		if line != "" {
			// Log only: the terminal already shows the typed line.
			c.logLine(line)
		}
		out, quit := c.HandleLine(line)
		if out != "" {
			c.emit(out)
		}
		if quit {
			return nil
		}
	}
}

func (c *Console) emit(s string) {
	fmt.Fprintln(c.Out, s)
	if c.LogFile != nil {
		fmt.Fprintln(c.LogFile, s)
	}
}

// logLine records an accepted command line in the session log.
func (c *Console) logLine(line string) {
	if c.LogFile != nil {
		fmt.Fprintf(c.LogFile, "> %s\n", line)
	}
}

func (c *Console) usage() string {
	return "commands:\n" + c.completer.Tree("    ")
}

// newCompleter builds tab completion: built-ins plus the mode's command table.
func newCompleter(mode protocol.Mode) *readline.PrefixCompleter {
	table := cmdTable(mode)
	pc := readline.NewPrefixCompleter(
		readline.PcItem("quit"),
		readline.PcItem("help"),
		readline.PcItem("auth"),
		readline.PcItem("errinfo"),
		readline.PcItem("cmdinfo"),
	)
	for name, meta := range table {
		item := readline.PcItem(name)
		if meta.subs != "" {
			for sc := range strings.SplitSeq(meta.subs, ",") {
				item.Children = append(item.Children, readline.PcItem(sc))
			}
		}
		pc.Children = append(pc.Children, item)
	}
	return pc
}

func cmdTable(mode protocol.Mode) map[string]cmdMeta {
	if mode == protocol.ModeCXRF {
		return intCmd
	}
	return extCmd
}
