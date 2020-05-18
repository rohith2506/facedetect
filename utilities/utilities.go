package utilities

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"math/rand"
	"os"
)

const (
	letterBytes  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	maxLength    = 30
	maxHashBytes = 16
)

// RandStringBytes ...
func RandStringBytes() string {
	b := make([]byte, maxLength)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// Find ...
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

// GetImageHash ...
func GetImageHash(imagePath string) (string, error) {
	var md5Hash string

	file, err := os.Open(imagePath)
	if err != nil {
		return md5Hash, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return md5Hash, err
	}
	hashInBytes := hash.Sum(nil)[:maxHashBytes]
	md5Hash = hex.EncodeToString(hashInBytes)
	return md5Hash, nil
}
