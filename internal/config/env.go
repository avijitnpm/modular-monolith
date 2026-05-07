package config

import (
	"log"

	"github.com/joho/godotenv"
)

func loadEnvFile() {
	err := godotenv.Load()

	if err != nil {
		log.Println(".env file not found, using system env")
	}
}
