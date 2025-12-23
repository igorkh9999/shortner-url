package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashVisitor creates a SHA256 hash of IP address and user agent
// This is used to identify unique visitors
func HashVisitor(ip, userAgent string) string {
	data := ip + userAgent
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

