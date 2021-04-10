package main

import (
	"fmt"
	"log"
)

func (sc syscon) sendCXRCommand(cmd string) {
	checksum := countChecksum(cmd)
	fcmd := fmt.Sprintf("C:%02X:%s", checksum, cmd)
	maxSize := 15 // for some reason we need to split every 15 symbols
	length := len(fcmd)
	var j int
	for i := 0; i < length; i += maxSize {
		j += maxSize
		if j > length {
			// j = length
			break // And don'r send extra
		}
		fmt.Println(fcmd[i:j])
		sc.writeCommand(fcmd[i:j])
	}
	sc.writeCommand("\r\n")
}

func (sc syscon) receiveCXRCommand() string {
	// buff := make([]byte, 1000)
	// n, err := sc.port.Read(buff)
	// fmt.Printf("Read %v bytes\n", n)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(string(buff[:n]))

	// // test := strings.SplitAfterN(string(buff[:n]), "\n", 2)
	// reg, err := regexp.Compile("[^a-zA-Z0-9:]+")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// processedString := reg.ReplaceAllString(string(buff[:n]), "")
	// fmt.Println(processedString)
	// return processedString
	buff := make([]byte, 1000)
	for {
		n, err := sc.port.Read(buff)
		fmt.Printf("Read %v bytes\n", n)
		if err != nil {
			log.Fatal(err)
		}
		if n == 0 {
			fmt.Println("\nEOF")
			break
		}

		// if strings.Contains(string(buff[:n]), "\r\n") {
		// 	test := strings.SplitAfterN(string(buff[:n]), "\n", 2)
		fmt.Println(string(buff[:n]))
		// 	return strings.TrimSpace(test[1])
		// }
	}
	return ""
}
