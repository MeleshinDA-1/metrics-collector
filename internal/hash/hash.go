package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

const Header = "HashSHA256"

func Sign(data []byte, key string) string {
	return base64.StdEncoding.EncodeToString(sum(data, key))
}

func Equal(data []byte, key string, signature string) bool {
	expected, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	return hmac.Equal(sum(data, key), expected)
}

func sum(data []byte, key string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)

	return mac.Sum(nil)
}
