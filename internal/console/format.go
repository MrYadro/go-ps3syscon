package console

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MrYadro/go-ps3syscon/internal/protocol"
)

// formatResult renders a Result per mode (mirrors the Python tool's output).
func formatResult(mode protocol.Mode, res protocol.Result) string {
	switch mode {
	case protocol.ModeCXR:
		return fmt.Sprintf("%08X %s", res.Code, strings.Join(res.Lines, " "))
	case protocol.ModeSW:
		if len(res.Lines) == 0 {
			return fmt.Sprintf("%08X", res.Code)
		}
		return fmt.Sprintf("%08X\n%s", res.Code, strings.Join(res.Lines, "\n"))
	default: // ModeCXRF
		return res.Raw
	}
}

func (c *Console) errinfo(arg string) string {
	if arg == "" {
		return "Please provide error code!"
	}
	return parseErrorCode(arg)
}

var errCodeRe = regexp.MustCompile(`^0xa[A-Fa-f0-9]{3}[1-4][0-9][0-6f][0-9f]$`)

func parseErrorCode(err string) string {
	if !errCodeRe.MatchString(err) {
		return "Unknown error!"
	}
	stepNo := err[4:6]
	errCat := err[6:7]
	errNo := err[6:10]
	return fmt.Sprintf("%s on step %s with error info: %s", scErrors[errCat], stepNo, scErrors[errNo])
}

func (c *Console) cmdinfo(arg string) string {
	cm, ok := cmdTable(c.Mode)[arg]
	if !ok {
		return "Wrong command"
	}
	params, subs := "no", "no"
	if cm.params != "" {
		params = strings.ReplaceAll(cm.params, ",", ", ")
	}
	if cm.subs != "" {
		subs = strings.ReplaceAll(cm.subs, ",", ", ")
	}
	return fmt.Sprintf("%s - %s, command called with %s parametres and %s subcommands", arg, cm.description, params, subs)
}
