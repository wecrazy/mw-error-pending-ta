package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Database
	DBUser string
	DBPass string
	DBHost string
	DBPort string
	DBName string

	// Paths & URLs
	MainPath      string
	OdooLoginURL  string
	OdooGetURL    string
	OdooUpdateURL string
	FileStoreURL  string
	FileStoreURL1 string
	WegilURL      string

	// Server
	ServerPort string
}

// Load reads the .env file and returns a populated Config.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env: %w", err)
	}

	return &Config{
		DBUser:        os.Getenv("DB_USER"),
		DBPass:        os.Getenv("DB_PASS"),
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        os.Getenv("DB_PORT"),
		DBName:        os.Getenv("DB_NAME"),
		MainPath:      os.Getenv("DATA_PATH"),
		OdooLoginURL:  os.Getenv("ODOO_LOGIN_URL"),
		OdooGetURL:    os.Getenv("ODOO_GET_URL"),
		OdooUpdateURL: os.Getenv("ODOO_UPDATE_URL"),
		FileStoreURL:  os.Getenv("FILESTORE_URL"),
		FileStoreURL1: os.Getenv("FILESTORE_FILE_URL"),
		WegilURL:      os.Getenv("WA_WEGIL_URL"),
		ServerPort:    os.Getenv("SERVER_PORT"),
	}, nil
}
