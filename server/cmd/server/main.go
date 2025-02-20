package main

import (
	"fmt"
	"fundamental/server/config"
	"fundamental/server/internal/api"
	"fundamental/server/internal/database"
	"fundamental/server/internal/geocoding"
	"fundamental/server/internal/models"
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

	// Initialize scheduler with supported cities
	scheduler := scheduler.NewScheduler(spiderManager, logger, config.GetCityNames())

	// Start scheduler
	scheduler.Start()
	logger.Info("Started scheduler for automated scraping")

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

// migrateMetropolitanAreas migrates metropolitan areas from JSON to the database
func migrateMetropolitanAreas(db *database.Database, logger *logrus.Logger) error {
	// Get areas from JSON config
	areas := config.GetMetroAreas()

	// Migrate each area to the database
	for _, area := range areas {
		// Convert to database model
		dbArea := models.MetropolitanArea{
			Name:   area.Name,
			Cities: area.Cities,
		}

		// Save to database
		if err := db.UpdateMetroArea(dbArea); err != nil {
			return fmt.Errorf("failed to migrate area %s: %v", area.Name, err)
		}

		logger.Infof("Migrated metropolitan area: %s", area.Name)
	}

	return nil
}
