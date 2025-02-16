package main

import (
	"fundamental/server/internal/api"
	"fundamental/server/internal/database"
	"fundamental/server/internal/geocoding"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		logger.WithError(err).Fatal("Failed to get current directory")
	}

	// Construct database path relative to the server directory
	dbPath := filepath.Join(currentDir, "database", "funda.db")
	logger.Infof("Using database at: %s", dbPath)

	// Initialize database
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize database")
	}
	defer db.Close()

	// Run database migrations
	logger.Info("Running database migrations...")
	if err := db.RunMigrations(); err != nil {
		logger.WithError(err).Fatal("Failed to run database migrations")
	}

	// Initialize geocoder
	cacheDir := filepath.Join(os.TempDir(), "fundamental", "geocode_cache")
	geocoder := geocoding.NewGeocoder(logger, cacheDir)

	// Start geocoding in a background goroutine
	go func() {
		logger.Info("Starting initial geocoding of properties without coordinates in background...")
		if err := db.UpdateMissingCoordinates(geocoder); err != nil {
			logger.WithError(err).Error("Failed to update coordinates")
		}
	}()

	// Initialize router
	router := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3004"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	router.Use(cors.New(config))

	// Setup API routes
	api.SetupRoutes(router, db)

	// Use port 5250
	const port = "5250"
	logger.Infof("Starting server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		logger.WithError(err).Fatal("Server failed to start")
	}
}
