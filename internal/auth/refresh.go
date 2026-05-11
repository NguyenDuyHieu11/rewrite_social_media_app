package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

var bufSize int = 32

func GenerateRefreshToken() (string, string, error) {
	buf := make([]byte, bufSize)
	_, err := rand.Read(buf)
	if err != nil {
		return "", "", fmt.Errorf("fail at rand.Read(buf): %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(buf)
	hash := HashRefreshToken(token)
	return token, hash, nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
