package api

import (
	"fmt"
	"fundamental/server/internal/database"
	"fundamental/server/internal/geocoding"
	"fundamental/server/internal/geometry"
	"fundamental/server/internal/models"
	"fundamental/server/internal/scraping"
	"fundamental/server/internal/telegram"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	db              *database.Database
	logger          *logrus.Logger
	geocoder        *geocoding.Geocoder
	districtManager *geometry.DistrictManager
	spiderManager   *scraping.SpiderManager
	telegramService *telegram.Service
}

type DateRange struct {
	StartDate string `form:"startDate"`
	EndDate   string `form:"endDate"`
}

type SpiderRequest struct {
	Place    string `json:"place" binding:"required"`
	MaxPages *int   `json:"max_pages"`
	Resume   bool   `json:"resume"`
}

func NewHandler(db *database.Database, logger *logrus.Logger) *Handler {
	if logger == nil {
		logger = logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetOutput(os.Stdout)
	}

	cacheDir := filepath.Join(os.TempDir(), "fundamental", "geocode_cache")

	// Initialize the district manager
	districtManager := geometry.NewDistrictManager(db.GetDB(), logger)

	// Initialize the spider manager
	spiderManager := scraping.NewSpiderManager(db, logger)

	// Initialize the telegram service
	telegramService := telegram.NewService(logger)

	// Load existing Telegram configuration
	if config, err := db.GetTelegramConfig(); err == nil && config != nil {
		telegramService.UpdateConfig(config)
	}

	return &Handler{
		db:              db,
		logger:          logger,
		geocoder:        geocoding.NewGeocoder(logger, cacheDir),
		districtManager: districtManager,
		spiderManager:   spiderManager,
		telegramService: telegramService,
	}
}

func (h *Handler) GetAllProperties(c *gin.Context) {
	var dateRange DateRange
	if err := c.ShouldBindQuery(&dateRange); err != nil {
		h.logger.WithError(err).Error("Failed to parse date range")
	}

	city := c.Query("city")
	properties, err := h.db.GetAllProperties(dateRange.StartDate, dateRange.EndDate, city)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get properties")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get properties"})
		return
	}

	c.JSON(http.StatusOK, properties)
}

func (h *Handler) GetPropertyStats(c *gin.Context) {
	var dateRange DateRange
	if err := c.ShouldBindQuery(&dateRange); err != nil {
		h.logger.WithError(err).Error("Failed to parse date range")
	}

	city := c.Query("city")
	stats, err := h.db.GetPropertyStats(dateRange.StartDate, dateRange.EndDate, city)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get property stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get property stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetAreaStats(c *gin.Context) {
	postalPrefix := c.Param("postal_prefix")
	var dateRange DateRange
	if err := c.ShouldBindQuery(&dateRange); err != nil {
		h.logger.WithError(err).Error("Failed to parse date range")
	}

	city := c.Query("city")
	stats, err := h.db.GetAreaStats(postalPrefix, dateRange.StartDate, dateRange.EndDate, city)
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

	var dateRange DateRange
	if err := c.ShouldBindQuery(&dateRange); err != nil {
		h.logger.WithError(err).Error("Failed to parse date range")
	}

	city := c.Query("city")
	sales, err := h.db.GetRecentSales(limit, dateRange.StartDate, dateRange.EndDate, city)
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

func (h *Handler) UpdateDistrictHulls(c *gin.Context) {
	err := h.districtManager.UpdateDistrictHulls()
	if err != nil {
		h.logger.WithError(err).Error("Failed to update district hulls")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update district hulls"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "District hulls updated successfully",
	})
}

func (h *Handler) RunActiveSpider(c *gin.Context) {
	var req SpiderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse spider request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	err := h.spiderManager.RunActiveSpider(req.Place, req.MaxPages)
	if err != nil {
		h.logger.WithError(err).Error("Failed to run active spider")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run spider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Active spider started successfully",
	})
}

func (h *Handler) RunSoldSpider(c *gin.Context) {
	var req SpiderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse spider request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	err := h.spiderManager.RunSoldSpider(req.Place, req.MaxPages, req.Resume)
	if err != nil {
		h.logger.WithError(err).Error("Failed to run sold spider")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run spider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Sold spider started successfully",
	})
}

// GetTelegramConfig returns the current Telegram configuration
func (h *Handler) GetTelegramConfig(c *gin.Context) {
	config, err := h.db.GetTelegramConfig()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get Telegram config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Telegram config"})
		return
	}

	if config == nil {
		c.JSON(http.StatusOK, gin.H{
			"is_enabled": false,
			"chat_id":    "",
			"bot_token":  "",
		})
		return
	}

	// Don't send the full bot token back to the client for security
	config.BotToken = "••••" + config.BotToken[len(config.BotToken)-4:]
	c.JSON(http.StatusOK, config)
}

// UpdateTelegramConfig updates the Telegram configuration
func (h *Handler) UpdateTelegramConfig(c *gin.Context) {
	var request models.TelegramConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Test the Telegram configuration before saving
	testService := telegram.NewService(h.logger)
	testConfig := &models.TelegramConfig{
		BotToken:  request.BotToken,
		ChatID:    request.ChatID,
		IsEnabled: true,
	}
	testService.UpdateConfig(testConfig)

	testMessage := "🔔 Test notification from FundaMental\n\nIf you see this message, your Telegram configuration is working correctly!"
	if err := testService.SendMessage(testMessage); err != nil {
		h.logger.WithError(err).Error("Failed to send test message")
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to send test message: %v", err)})
		return
	}

	// Save the configuration
	if err := h.db.UpdateTelegramConfig(&request); err != nil {
		h.logger.WithError(err).Error("Failed to update Telegram config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Telegram config"})
		return
	}

	// Update the service configuration
	if config, err := h.db.GetTelegramConfig(); err == nil && config != nil {
		h.telegramService.UpdateConfig(config)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Telegram configuration updated successfully"})
}
