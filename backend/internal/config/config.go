package config

import (
	"bufio"
	"os"
	"strings"
)

func init() {
	loadEnvFile(".env")
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}

type Config struct {
	RPCUrl          string
	ChainID         string
	PrivateKey      string
	ContractAddress string
	TokenAAddress   string
	TokenBAddress   string
	DatabaseURL     string
	RedisURL        string
	ServerPort      string
}

func Load() *Config {
	return &Config{
		RPCUrl:          getEnv("RPC_URL", "http://localhost:8545"),
		ChainID:         getEnv("CHAIN_ID", "31337"),
		PrivateKey:      getEnv("PRIVATE_KEY", ""),
		ContractAddress: getEnv("NEXUS_CONTRACT_ADDRESS", ""),
		TokenAAddress:   getEnv("TOKEN_A_ADDRESS", ""),
		TokenBAddress:   getEnv("TOKEN_B_ADDRESS", ""),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://nexus:nexus_dev@localhost:5432/nexus_orderbook?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "localhost:6379"),
		ServerPort:      getEnv("SERVER_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
