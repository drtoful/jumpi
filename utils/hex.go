package utils

import (
	"encoding/hex"
)

// we put this into utils, in case we want to change it
// in the future

func Hexlify(data []byte) string {
	return hex.EncodeToString(data)
}

func UnHexlify(data string) ([]byte, error) {
	return hex.DecodeString(data)
}
