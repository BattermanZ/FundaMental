package main

import (
	"fundamental/server/internal/api"
	"fundamental/server/internal/database"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// Get the project root directory
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(os.Args[0])))
	dbPath := filepath.Join(projectRoot, "database", "funda.db")

	// Initialize database
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize database")
	}
	defer db.Close()

	// Initialize handler
	handler := api.NewHandler(db, logger)

	// Initialize router
	router := mux.NewRouter()

	// Apply CORS middleware
	router.Use(handler.EnableCORS)

	// Define routes
	router.HandleFunc("/api/properties", handler.GetAllProperties).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/stats", handler.GetPropertyStats).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/areas/{postal_prefix}", handler.GetAreaStats).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/recent-sales", handler.GetRecentSales).Methods("GET", "OPTIONS")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Infof("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.WithError(err).Fatal("Server failed to start")
	}
}
