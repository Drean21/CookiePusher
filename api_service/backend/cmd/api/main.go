package main

import (
	"cookie-syncer/api/internal/router"
	"cookie-syncer/api/internal/store/sqlitestore"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite" // Import the pure Go SQLite driver
)

const dbFileName = "cookiesyncer.db"

func main() {
	// Configure zerolog for pretty console output
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Initialize a new SQLite store.
	db, err := sqlitestore.New(dbFileName)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	log.Info().Msgf("Database initialized and connected at %s", dbFileName)

	// Create a new router and pass the store to it.
	mux := router.NewRouter(db)
	
	// Print all registered routes
	router.PrintRoutes(mux)

	log.Info().Msg("Starting API server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal().Err(err).Msg("Could not start server")
	}
}
