package config

import (
	"flag"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// Config holds all configuration for the application.
type Config struct {
	PoolAccessKey string
	AdminKey      string
	DBType        string // "sqlite", "postgres", or "mysql"
	DSN           string // Data Source Name for the database
}

// Load loads the configuration from .env file, environment variables, and command-line flags.
func Load() *Config {
	// Attempt to load .env file, but don't treat it as an error if it doesn't exist.
	if err := godotenv.Load(); err != nil {
		log.Info().Msg("No .env file found, will rely on environment variables and flags.")
	}

	var cfg Config

	// Define command-line flags
	flag.StringVar(&cfg.PoolAccessKey, "pool-key", "", "Access key for the cookie pool API")
	flag.StringVar(&cfg.AdminKey, "admin-key", "", "Key for accessing admin endpoints")
	flag.StringVar(&cfg.DBType, "db-type", "sqlite", "Database type (sqlite, postgres, mysql)")
	flag.StringVar(&cfg.DSN, "dsn", "CookiePusher.db", "Database connection string (DSN)")
	flag.Parse()

	// If a value was not set by a flag, try to get it from the environment variable.
	if cfg.PoolAccessKey == "" {
		cfg.PoolAccessKey = os.Getenv("POOL_ACCESS_KEY")
	}
	if cfg.AdminKey == "" {
		cfg.AdminKey = os.Getenv("ADMIN_KEY")
	}
	if dbTypeFromEnv := os.Getenv("DB_TYPE"); dbTypeFromEnv != "" {
		cfg.DBType = dbTypeFromEnv
	}
	if dsnFromEnv := os.Getenv("DSN"); dsnFromEnv != "" {
		cfg.DSN = dsnFromEnv
	}

	return &cfg
}
