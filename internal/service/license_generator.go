package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const defaultCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func ParseCharset(input string) (string, error) {
	if input == "" {
		return defaultCharset, nil
	}
	var builder strings.Builder
	parts := strings.Split(input, ",")
	for _, part := range parts {
		if len(part) == 3 && part[1] == '-' {
			start := part[0]
			end := part[2]
			if start > end {
				return "", fmt.Errorf("invalid range: %s", part)
			}
			for i := start; i <= end; i++ {
				builder.WriteByte(i)
			}
		} else {
			builder.WriteString(part)
		}
	}
	return builder.String(), nil
}

func GenerateLicenseKey(prefix string, length int, separator string, charset string) (string, error) {
	if length <= 0 {
		length = 12 // Default length
	}

	if separator == "" {
		separator = "-"
	}

	usedCharset := defaultCharset
	if charset != "" {
		usedCharset = charset
	}

	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(usedCharset))))
		if err != nil {
			return "", err
		}
		b[i] = usedCharset[num.Int64()]
	}

	randomString := string(b)
	if prefix != "" {
		return prefix + separator + randomString, nil
	}
	return randomString, nil
}
