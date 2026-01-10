package main

import (
	"flag"
	"fmt"
	"log"

	"clortho/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadFromPath(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{})

	tokenString, err := token.SignedString([]byte(cfg.AdminSecret))
	if err != nil {
		log.Fatalf("Error signing token: %v", err)
	}

	fmt.Println(tokenString)
}
