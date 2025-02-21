package main

import (
	"fundamental/server/config"
	"fundamental/server/internal/api"
	"fundamental/server/internal/database"
	"fundamental/server/internal/geocoding"
	"fundamental/server/internal/scheduler"
	"fundamental/server/internal/scraping"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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

	// Initialize spider manager
	spiderManager := scraping.NewSpiderManager(db, logger)

	// Initialize scheduler with cities from database
	cityNames, err := config.GetCityNames(db)
	if err != nil {
		logger.WithError(err).Fatal("Failed to get city names for scheduler")
	}
	// Note: GetCityNames returns normalized city names suitable for Funda URLs
	scheduler := scheduler.NewScheduler(spiderManager, logger, cityNames)

	// Comment out scheduler auto-start - uncomment when needed
	// scheduler.Start()
	// logger.Info("Started scheduler for automated scraping")

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
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:3004"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type"}
	router.Use(cors.New(corsConfig))

	// Setup API routes
	api.SetupRoutes(router, db)
	api.SetupMetropolitanRoutes(router, db, geocoder)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Info("Shutting down scheduler...")
		scheduler.Stop()
		logger.Info("Scheduler stopped")
		os.Exit(0)
	}()

	// Use port 5250
	const port = "5250"
	logger.Infof("Starting server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		logger.WithError(err).Fatal("Server failed to start")
	}
}
