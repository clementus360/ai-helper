package config

import (
	"github.com/joho/godotenv"
)

// Load environment variables and handle errors

func LoadEnv() {
	err := godotenv.Load()

	if err != nil {
		Logger.Warn("Error loading .env file, will use environment variables instead:", err)
		// Don't call Fatal here - continue execution
	}
}
