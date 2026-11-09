package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
)

func epsHash(value, hashKey string) string {
	mac := hmac.New(sha512.New, []byte(hashKey))
	_, _ = mac.Write([]byte(value))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
