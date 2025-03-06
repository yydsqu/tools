package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

var (
	Hex = hexEncode{}
)

type Encode interface {
	EncodeToString(src []byte) string
	DecodeString(s string) ([]byte, error)
}

type CFBCipher struct {
	cipher.Block
	Encode
}

func (c *CFBCipher) Encrypt(src []byte) (string, error) {
	var err error
	ciphertext := make([]byte, aes.BlockSize+len(src))
	iv := ciphertext[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	stream := cipher.NewCFBEncrypter(c.Block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], src)
	return c.EncodeToString(ciphertext), nil
}

func (c *CFBCipher) Decrypt(raw string) ([]byte, error) {
	src, err := c.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	if len(src) < aes.BlockSize {
		return nil, fmt.Errorf("cipher text is too short")
	}
	iv := src[:aes.BlockSize]
	src = src[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(c.Block, iv)
	stream.XORKeyStream(src, src)
	return src, nil
}

func NewCFBCipher(key []byte, encode Encode) (*CFBCipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if encode == nil {
		encode = Hex
	}
	return &CFBCipher{
		Block:  block,
		Encode: encode,
	}, nil
}

type CBCCipher struct {
	cipher.Block
	Encode
}

func (c *CBCCipher) PKCS7Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

func (c *CBCCipher) UPKCS7Padding(src []byte) []byte {
	n := len(src)
	if n == 0 {
		return src
	}
	paddingNum := int(src[n-1])
	return src[:n-paddingNum]
}

func (c *CBCCipher) Encrypt(src []byte) (string, error) {
	var err error
	src = c.PKCS7Padding(src, aes.BlockSize)
	ciphertext := make([]byte, aes.BlockSize+len(src))
	iv := ciphertext[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	stream := cipher.NewCBCEncrypter(c.Block, iv)
	stream.CryptBlocks(ciphertext[aes.BlockSize:], src)
	return c.EncodeToString(ciphertext), nil
}

func (c *CBCCipher) Decrypt(raw string) ([]byte, error) {
	src, err := c.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	if len(src) < aes.BlockSize {
		return nil, fmt.Errorf("cipher text is too short")
	}
	iv := src[:aes.BlockSize]
	src = src[aes.BlockSize:]
	stream := cipher.NewCBCDecrypter(c.Block, iv)
	stream.CryptBlocks(src, src)
	return c.UPKCS7Padding(src), nil
}

func NewCBCCipher(key []byte, encode Encode) (*CBCCipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if encode == nil {
		encode = Hex
	}
	return &CBCCipher{
		Block:  block,
		Encode: encode,
	}, nil
}

type hexEncode struct {
}

func (h hexEncode) EncodeToString(src []byte) string {
	return hex.EncodeToString(src)
}

func (h hexEncode) DecodeString(s string) ([]byte, error) {
	return hex.DecodeString(s)
}
