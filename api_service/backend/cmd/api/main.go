package main

import (
	"cookie-syncer/api/internal/config"
	"cookie-syncer/api/internal/handler"
	"cookie-syncer/api/internal/router"
	"cookie-syncer/api/internal/store/gormstore"
	"fmt"
	"net/http"
	"os"

	"cookie-syncer/api/docs" // Import the generated docs

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// @title           Cookie Syncer API
// @version         1.0
// @description     This is the API server for the Cookie Syncer browser extension.
// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apiKey ApiKeyAuth
// @in              header
// @name            x-api-key
//
// @securityDefinitions.apiKey PoolKeyAuth
// @in              header
// @name            x-pool-key
//
// @securityDefinitions.apiKey AdminKeyAuth
// @in              header
// @name            x-admin-key
func main() {
	// Configure zerolog for pretty console output
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load application configuration
	cfg := config.Load()

	// Set log level based on configuration
	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Warn().Msgf("Invalid log level '%s', defaulting to 'info'", cfg.LogLevel)
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// Initialize a new GORM store based on configuration.
	db, err := gormstore.New(cfg, cfg.AdminKey, cfg.PoolAccessKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	log.Info().Msgf("Database initialized and connected to %s database", cfg.DBType)

	// Create a new router and pass the store to it.
	lockManager := handler.NewUserLockManager()
	mux := router.NewRouter(db, lockManager, cfg)

	// Print all registered routes
	router.PrintRoutes(mux)

	// Dynamically set swagger host.
	// Use SwaggerHost if provided, otherwise fall back to the server's host and port.
	if cfg.SwaggerHost != "" {
		docs.SwaggerInfo.Host = cfg.SwaggerHost
		// When using a public host, we assume HTTPS is handled by a proxy.
		log.Info().Msgf("Swagger UI is available at https://%s/swagger/index.html", docs.SwaggerInfo.Host)
	} else {
		docs.SwaggerInfo.Host = fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
		log.Info().Msgf("Swagger UI is available at http://%s/swagger/index.html", docs.SwaggerInfo.Host)
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	log.Info().Msgf("Starting API server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal().Err(err).Msg("Could not start server")
	}
}
