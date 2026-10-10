package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// magic aes key, don't edit
var magicKey = [32]byte{
	141, 213, 179, 33, 189, 188, 61, 90,
	233, 41, 129, 156, 195, 244, 208, 161,
	164, 72, 170, 198, 88, 73, 63, 165,
	85, 86, 200, 113, 172, 103, 224, 57,
}

func Encrypt(plaintext []byte) (string, error) {
	c, err := aes.NewCipher(magicKey[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, plaintext, nil)), nil
}

func Decrypt(cipherstr string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cipherstr)
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(magicKey[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
