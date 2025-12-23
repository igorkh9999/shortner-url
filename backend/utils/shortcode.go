package utils

import (
	"crypto/rand"
)

const (
	shortCodeLength = 6
	charset        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// GenerateShortCode generates a random 6-character alphanumeric short code
func GenerateShortCode() string {
	bytes := make([]byte, shortCodeLength)
	rand.Read(bytes)
	
	code := make([]byte, shortCodeLength)
	for i := 0; i < shortCodeLength; i++ {
		code[i] = charset[bytes[i]%byte(len(charset))]
	}
	
	return string(code)
}

