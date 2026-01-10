package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	var publicKeyB64, tokenString string

	flag.StringVar(&publicKeyB64, "pubkey", "", "Base64 encoded public key (from config.yaml)")
	flag.StringVar(&tokenString, "token", "", "JWT Token (from API response)")
	
	flag.Parse()

	if publicKeyB64 == "" || tokenString == "" {
		fmt.Println("Usage: go run scripts/verify_token.go -pubkey <...> -token <...>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Decode Public Key
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		fmt.Printf("Error decoding public key: %v\n", err)
		os.Exit(1)
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		fmt.Printf("Invalid public key size: %d\n", len(pubKeyBytes))
		os.Exit(1)
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)

	// Parse and Verify Token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})

	if err != nil {
		fmt.Printf("❌ Token validation failed: %v\n", err)
		os.Exit(1)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		fmt.Println("✅ Token is VALID and AUTHENTIC.")
		
		fmt.Println("\nLicense Details:")
		fmt.Printf("- Key: %s\n", claims["sub"])
		fmt.Printf("- Valid: %v\n", claims["valid"])
		fmt.Printf("- Features: %v\n", claims["features"])
		
		if exp, ok := claims["exp"].(float64); ok {
			tm := time.Unix(int64(exp), 0)
			fmt.Printf("- Expires: %s\n", tm.Format(time.RFC3339))
			
			if time.Now().After(tm) {
				fmt.Println("❌ LICENSE EXPIRED")
			} else {
				fmt.Println("✅ LICENSE ACTIVE")
			}
		} else {
			fmt.Println("- Expires: Never (Perpetual)")
		}

	} else {
		fmt.Println("❌ Token is INVALID.")
	}
}
