package main

import (
	"crypto/aes"
	"crypto/cipher"
)

func countChecksum(cmd string) byte {
	var sum byte
	cmdBytes := []byte(cmd)
	for _, v := range cmdBytes {
		sum += v
	}
	return sum
}

func decode(ciphertext []byte) []byte {
	block, err := aes.NewCipher(sc2TBKey)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, zero)

	mode.CryptBlocks(ciphertext, ciphertext)

	return ciphertext
}

func encode(plaintext []byte) []byte {
	if len(plaintext)%aes.BlockSize != 0 {
		panic("plaintext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(tb2SCKey)
	if err != nil {
		panic(err)
	}

	ciphertext := make([]byte, len(plaintext))

	mode := cipher.NewCBCEncrypter(block, zero)
	mode.CryptBlocks(ciphertext, plaintext)

	return ciphertext
}
