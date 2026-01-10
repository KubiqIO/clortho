package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port                      string          `yaml:"port"`
	Debug                     bool            `yaml:"debug"`
	DatabaseURL               string          `yaml:"database_url"`
	AdminSecret               string          `yaml:"admin_secret"`
	ResponseSigningPrivateKey string          `yaml:"response_signing_private_key"`
	ResponseSigningPublicKey  string          `yaml:"response_signing_public_key"`
	TrustedProxies            []string        `yaml:"trusted_proxies"`
	RateLimitAdmin            RateLimitConfig `yaml:"rate_limit_admin"`
	RateLimitCheck            RateLimitConfig `yaml:"rate_limit_check"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64       `yaml:"requests_per_second"`
	Burst             int           `yaml:"burst"`
	Enabled           bool          `yaml:"enabled"`
	CacheSize         int           `yaml:"cache_size"`
	CacheTTL          time.Duration `yaml:"cache_ttl"`
}

func Load() (Config, error) {
	return LoadFromPath("config.yaml")
}

func LoadFromPath(path string) (Config, error) {
	cfg := NewDefaultConfig()

	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		decoder := yaml.NewDecoder(f)
		if err := decoder.Decode(&cfg); err != nil {
			return cfg, err
		}
	} else if !os.IsNotExist(err) {
		return cfg, err
	}

	if err := cfg.ensureKeys(); err != nil {
		return cfg, err
	}

	if err := cfg.ensureAdminSecret(); err != nil {
		return cfg, err
	}

	cfg.LoadEnv()

	return cfg, nil
}

func NewDefaultConfig() Config {
	return Config{
		Port:  "8080",
		Debug: false,
		RateLimitAdmin: RateLimitConfig{
			RequestsPerSecond: 5,
			Burst:             10,
			Enabled:           true,
			CacheSize:         5000,
			CacheTTL:          1 * time.Hour,
		},
		RateLimitCheck: RateLimitConfig{
			RequestsPerSecond: 5,
			Burst:             10,
			Enabled:           true,
			CacheSize:         5000,
			CacheTTL:          1 * time.Hour,
		},
	}
}

func (c *Config) LoadEnv() {
	if envPort := os.Getenv("PORT"); envPort != "" {
		c.Port = envPort
	}
	if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
		c.DatabaseURL = envDB
	}
	if envSecret := os.Getenv("ADMIN_SECRET"); envSecret != "" {
		c.AdminSecret = envSecret
	}
	if envPrivKey := os.Getenv("RESPONSE_SIGNING_PRIVATE_KEY"); envPrivKey != "" {
		c.ResponseSigningPrivateKey = envPrivKey
	}
	if envPubKey := os.Getenv("RESPONSE_SIGNING_PUBLIC_KEY"); envPubKey != "" {
		c.ResponseSigningPublicKey = envPubKey
	}
}

func (c *Config) ensureKeys() error {
	if c.ResponseSigningPrivateKey != "" && c.ResponseSigningPublicKey != "" {
		return nil
	}

	slog.Warn("ResponseSigningPrivateKey or ResponseSigningPublicKey not found, generating ephemeral key pair. THESE KEYS WILL BE LOST ON RESTART.")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate keys: %w", err)
	}

	privBase64 := base64.StdEncoding.EncodeToString(priv)
	pubBase64 := base64.StdEncoding.EncodeToString(pub)

	c.ResponseSigningPrivateKey = privBase64
	c.ResponseSigningPublicKey = pubBase64

	return nil
}

func (c *Config) ensureAdminSecret() error {
	if c.AdminSecret != "" {
		return nil
	}

	slog.Warn("Admin Secret not found, generating a random ephemeral one. THIS SECRET WILL BE LOST ON RESTART.")

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return fmt.Errorf("failed to generate admin secret: %w", err)
	}
	secret := base64.StdEncoding.EncodeToString(secretBytes)
	c.AdminSecret = secret

	return nil
}

