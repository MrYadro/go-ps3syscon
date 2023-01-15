package main

import (
	"fmt"
	"regexp"
	"strings"
)

func (sc syscon) VirtualCommandErrinfo(cmd string) string {
	errCode := strings.Split(cmd, " ")
	if len(errCode) < 2 {
		return "Please provide error code!"
	} else {
		return parseErrorCode(errCode[1])
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
