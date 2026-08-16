package protocol

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	// https://www.psdevwiki.com/ps3/Keys
	sc2TBKey            = []byte{113, 240, 63, 24, 76, 1, 197, 235, 195, 246, 162, 42, 66, 186, 149, 37} // 71F03F184C01C5EBC3F6A22A42BA9525
	tb2SCKey            = []byte{144, 126, 115, 15, 77, 78, 10, 11, 123, 117, 240, 48, 235, 29, 157, 54} // 907E730F4D4E0A0B7B75F030EB1D9D36
	auth1Response       = []byte{51, 80, 189, 120, 32, 52, 92, 41, 5, 106, 34, 59, 162, 32, 179, 35}     // 3350BD7820345C29056A223BA220B323
	zero                = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}                         // 00000000000000000000000000000000
	auth1ResponseHeader = []byte{16, 16, 0, 0, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0}               // 10100000FFFFFFFF0000000000000000
	auth2RequestHeader  = []byte{16, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}                        // 10010000000000000000000000000000
)

// authChallengeHex is the fixed AUTH1 challenge body (128 hex chars).
const authChallengeHex = "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

func decryptCBC(key, ciphertext []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err) // keys are fixed 16-byte constants
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, zero).CryptBlocks(plaintext, ciphertext)
	return plaintext
}

func encryptCBC(key, plaintext []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, zero).CryptBlocks(ciphertext, plaintext)
	return ciphertext
}

// parseAuth1Response validates a hex AUTH1 response and returns the hex
// AUTH2 request (header + encrypted body) to send back.
func parseAuth1Response(hexResp string) (string, error) {
	b, err := hex.DecodeString(strings.TrimSpace(hexResp))
	if err != nil {
		return "", fmt.Errorf("auth: bad hex: %w", err)
	}
	if len(b) != 0x40 {
		return "", errors.New("auth: wrong response length")
	}
	if !bytes.Equal(b[0:0x10], auth1ResponseHeader) {
		return "", errors.New("auth: wrong Auth1 response header")
	}
	data := decryptCBC(sc2TBKey, b[0x10:0x40])
	if !bytes.Equal(data[0x8:0x10], zero[0:0x8]) || !bytes.Equal(data[0x10:0x20], auth1Response) || !bytes.Equal(data[0x20:0x30], zero) {
		return "", errors.New("auth: wrong Auth1 response body")
	}
	newData := append(append([]byte{}, data[0x8:0x10]...), data[0x0:0x8]...)
	newData = append(newData, zero...)
	newData = append(newData, zero...)
	body := append(append([]byte{}, auth2RequestHeader...), encryptCBC(tb2SCKey, newData)...)
	return fmt.Sprintf("%02X", body), nil
}

// Auth performs the AUTH1/AUTH2 handshake for the connection's mode.
func (c *Conn) Auth() (string, error) {
	switch c.Mode {
	case ModeCXRF:
		return c.cxrfAuth()
	case ModeCXR, ModeSW:
		return c.cxrAuth()
	}
	return "", fmt.Errorf("auth: unknown mode %d", int(c.Mode))
}

// cxrAuth runs AUTH1/AUTH2 with CXR/SW framing (SW auto-issues SETCMDLONG
// for the long commands inside Command).
func (c *Conn) cxrAuth() (string, error) {
	res, err := c.Command("AUTH1 " + authChallengeHex)
	if err != nil {
		return "", fmt.Errorf("auth1: %w", err)
	}
	if res.Code != 0 || len(res.Lines) < 1 {
		return "", fmt.Errorf("auth1: unexpected response %q", res.Raw)
	}
	auth2, err := parseAuth1Response(res.Lines[0])
	if err != nil {
		return "", err
	}
	res, err = c.Command("AUTH2 " + auth2)
	if err != nil {
		return "", fmt.Errorf("auth2: %w", err)
	}
	if res.Code != 0 {
		return "Auth failed", nil
	}
	return "Auth successful", nil
}

// cxrfAuth wraps the handshake in scopen/SC_SUCCESS.
func (c *Conn) cxrfAuth() (string, error) {
	res, err := c.Command("scopen")
	if err != nil {
		return "", fmt.Errorf("scopen: %w", err)
	}
	if !strings.Contains(res.Raw, "SC_READY") {
		return fmt.Sprintf("Error opening syscon\n%s", res.Raw), nil
	}
	res, err = c.Command(authChallengeHex)
	if err != nil {
		return "", fmt.Errorf("auth1: %w", err)
	}
	auth2, err := parseAuth1Response(res.Raw)
	if err != nil {
		return "", err
	}
	res, err = c.Command(auth2)
	if err != nil {
		return "", fmt.Errorf("auth2: %w", err)
	}
	if !strings.Contains(res.Raw, "SC_SUCCESS") {
		return "Auth failed", nil
	}
	return "Auth successful", nil
}
