package config

import (
	"flag"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// Config holds all configuration for the application.
type Config struct {
	// Security
	PoolAccessKey string
	AdminKey      string

	// Database
	DBType               string // "sqlite", "postgres", or "mysql"
	DSN                  string // Data Source Name for the database
	DBMaxOpenConnections int
	DBMaxIdleConnections int

	// Server
	Port string
	Host string

	// Logging
	LogLevel     string
	GormLogLevel string
}

// Load loads the configuration from .env file, environment variables, and command-line flags.
func Load() *Config {
	// Attempt to load .env file, but don't treat it as an error if it doesn't exist.
	if err := godotenv.Load(); err != nil {
		log.Info().Msg("No .env file found, will rely on environment variables and flags.")
	}

	var cfg Config

	// Define command-line flags
	flag.StringVar(&cfg.PoolAccessKey, "pool-key", getEnv("POOL_ACCESS_KEY", ""), "Access key for the cookie pool API")
	flag.StringVar(&cfg.AdminKey, "admin-key", getEnv("ADMIN_KEY", ""), "Key for accessing admin endpoints")
	flag.StringVar(&cfg.DBType, "db-type", getEnv("DB_TYPE", "sqlite"), "Database type (sqlite, postgres, mysql)")
	flag.StringVar(&cfg.DSN, "dsn", getEnv("DSN", "CookiePusher.db"), "Database connection string (DSN)")
	flag.IntVar(&cfg.DBMaxOpenConnections, "db-max-open-conns", getEnvAsInt("DB_MAX_OPEN_CONNECTIONS", 25), "Database max open connections")
	flag.IntVar(&cfg.DBMaxIdleConnections, "db-max-idle-conns", getEnvAsInt("DB_MAX_IDLE_CONNECTIONS", 5), "Database max idle connections")
	flag.StringVar(&cfg.Port, "port", getEnv("PORT", "8080"), "Server port")
	flag.StringVar(&cfg.Host, "host", getEnv("HOST", "0.0.0.0"), "Server host")
	flag.StringVar(&cfg.LogLevel, "log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	flag.StringVar(&cfg.GormLogLevel, "gorm-log-level", getEnv("GORM_LOG_LEVEL", "silent"), "GORM log level (silent, info, warn, error)")

	flag.Parse()

	return &cfg
}

// Helper function to get an environment variable or return a default value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Helper function to get an environment variable as an integer or return a default value.
func getEnvAsInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}
