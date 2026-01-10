package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

)

func main() {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	privBase64 := base64.StdEncoding.EncodeToString(priv)
	pubBase64 := base64.StdEncoding.EncodeToString(pub)

	fmt.Printf("\nGenerated Ed25519 Keys:\n")
	fmt.Printf("----------------------------------------------------------------\n")
	fmt.Printf("ResponseSigningPrivateKey: %s\n", privBase64)
	fmt.Printf("ResponseSigningPublicKey:  %s\n", pubBase64)
	fmt.Printf("----------------------------------------------------------------\n")
	fmt.Printf("\nAdd these to your config.yaml or environment variables:\n")
	fmt.Printf("config.yaml:\n")
	fmt.Printf("  response_signing_private_key: \"%s\"\n", privBase64)
	fmt.Printf("  response_signing_public_key: \"%s\"\n", pubBase64)
	fmt.Printf("\nEnvironment Variables:\n")
	fmt.Printf("  RESPONSE_SIGNING_PRIVATE_KEY=%s\n", privBase64)
	fmt.Printf("  RESPONSE_SIGNING_PUBLIC_KEY=%s\n", pubBase64)
}
