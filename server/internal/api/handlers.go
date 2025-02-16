package api

import (
	"fundamental/server/internal/database"
	"fundamental/server/internal/geocoding"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	db       *database.Database
	logger   *logrus.Logger
	geocoder *geocoding.Geocoder
}

func NewHandler(db *database.Database, logger *logrus.Logger) *Handler {
	if logger == nil {
		logger = logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetOutput(os.Stdout)
	}

	cacheDir := filepath.Join(os.TempDir(), "fundamental", "geocode_cache")
	return &Handler{
		db:       db,
		logger:   logger,
		geocoder: geocoding.NewGeocoder(logger, cacheDir),
	}
}

func (h *Handler) GetAllProperties(c *gin.Context) {
	properties, err := h.db.GetAllProperties()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get properties")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get properties"})
		return
	}

	c.JSON(http.StatusOK, properties)
}

func (h *Handler) GetPropertyStats(c *gin.Context) {
	stats, err := h.db.GetPropertyStats()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get property stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get property stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetAreaStats(c *gin.Context) {
	postalPrefix := c.Param("postal_prefix")

	stats, err := h.db.GetAreaStats(postalPrefix)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get area stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get area stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetRecentSales(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	sales, err := h.db.GetRecentSales(limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get recent sales")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent sales"})
		return
	}

	c.JSON(http.StatusOK, sales)
}

func (h *Handler) UpdateCoordinates(c *gin.Context) {
	err := h.db.UpdateMissingCoordinates(h.geocoder)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update coordinates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update coordinates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "Coordinates update process started",
	})
}

// CORS middleware
func (h *Handler) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
