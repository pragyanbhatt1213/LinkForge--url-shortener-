package services

import (
	"crypto/rand"
	"math/big"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateSlug() string {
	slug := make([]byte, 6)
	for i := range slug {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		slug[i] = charset[n.Int64()]
	}
	return string(slug)
}
