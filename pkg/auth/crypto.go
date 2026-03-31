package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	ErrInvalidKey        = errors.New("encryption key must be 32 bytes")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrDecryptFailed     = errors.New("decryption failed")
)

// EncryptToken 使用 AES-256-GCM 加密 token
func EncryptToken(token string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密：nonce + ciphertext
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)

	// 返回 base64 编码的密文
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptToken 使用 AES-256-GCM 解密 token
func DecryptToken(encrypted string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKey
	}

	// Base64 解码
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	// 提取 nonce 和密文
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrDecryptFailed
	}

	return string(plaintext), nil
}
