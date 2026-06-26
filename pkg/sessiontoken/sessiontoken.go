package sessiontoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

var randRead = rand.Read

func New() (token string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := randRead(buf); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	token = hex.EncodeToString(buf)
	return token, Hash(token), nil
}

func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
