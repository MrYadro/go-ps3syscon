package main

func countChecksum(cmd string) byte {
	var sum byte
	cmdBytes := []byte(cmd)
	for _, v := range cmdBytes {
		sum += v
	}
	return sum
}
