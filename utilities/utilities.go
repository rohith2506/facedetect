package utilities

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
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
func GetImageHash(inputType int, response *http.Response, multipartFile *multipart.FileHeader) (string, error) {
	var md5Hash string
	if inputType == 1 && response == nil {
		return md5Hash, errors.New("empty image from url retrieval")
	}
	if inputType == 2 && multipartFile == nil {
		return md5Hash, errors.New("empty image from upload")
	}

	hash := md5.New()
	if inputType == 1 {
		if _, err := io.Copy(hash, response.Body); err != nil {
			return md5Hash, err
		}
	} else {
		file, err := os.Open(multipartFile.Filename)
		if err != nil {
			return md5Hash, err
		}
		defer file.Close()
		if _, err := io.Copy(hash, file); err != nil {
			return md5Hash, err
		}
	}

	hashInBytes := hash.Sum(nil)[:maxHashBytes]
	md5Hash = hex.EncodeToString(hashInBytes)
	return md5Hash, nil
}

//
