package crypto

import (
	"bytes"
	"fmt"
	"testing"
)

func PKCS7Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

func TestName(t *testing.T) {
	cipher, err := NewCBCCipher([]byte("1234567891234567"), Hex)
	if err != nil {
		t.Fatal(err)
	}
	encrypt, err := cipher.Encrypt([]byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	decrypt, err := cipher.Decrypt(encrypt)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(decrypt))
}
