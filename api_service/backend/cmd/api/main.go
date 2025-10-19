package main

import (
	"cookie-syncer/api/internal/config"
	"cookie-syncer/api/internal/handler"
	"cookie-syncer/api/internal/router"
	"cookie-syncer/api/internal/store/sqlitestore"
	"net/http"
	"os"

	"cookie-syncer/api/docs" // Import the generated docs

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite" // Import the pure Go SQLite driver
)

const dbFileName = "CookiePusher.db?_busy_timeout=5000"

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

	// Initialize a new SQLite store. The _busy_timeout parameter is crucial for handling concurrent requests.
	db, err := sqlitestore.New(dbFileName, cfg.AdminKey, cfg.PoolAccessKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	log.Info().Msgf("Database initialized and connected at %s", dbFileName)

	// Create a new router and pass the store to it.
	lockManager := handler.NewUserLockManager()
	mux := router.NewRouter(db, lockManager, cfg)
	
	// Print all registered routes
	router.PrintRoutes(mux)

	// Dynamically set swagger host
	docs.SwaggerInfo.Host = "localhost:8080"
	log.Info().Msgf("Swagger UI is available at http://%s/swagger/index.html", docs.SwaggerInfo.Host)

	log.Info().Msg("Starting API server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal().Err(err).Msg("Could not start server")
	}
}
