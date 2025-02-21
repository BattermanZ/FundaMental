package api

import (
	"fmt"
	"fundamental/server/config"
	"fundamental/server/internal/database"
	"fundamental/server/internal/geocoding"
	"fundamental/server/internal/geometry"
	"fundamental/server/internal/models"
	"fundamental/server/internal/scraping"
	"fundamental/server/internal/telegram"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	telegramService.SetDatabase(db)

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
	if err := c.ShouldBindJSON(&req); err != nil || req.Place == "" {
		// If no parameters provided or invalid JSON, use configured cities
		cities, err := config.GetCityNames(h.db)
		if err != nil {
			h.logger.WithError(err).Error("Failed to get configured cities")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get configured cities"})
			return
		}

		// Start spider for each configured city
		for _, city := range cities {
			normalizedCity := config.NormalizeCity(city)
			err := h.spiderManager.RunActiveSpider(normalizedCity, nil)
			if err != nil {
				h.logger.WithError(err).WithField("city", city).Error("Failed to run active spider")
				// Continue with other cities even if one fails
				continue
			}
			h.logger.WithField("city", city).Info("Started active spider successfully")
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Active spiders started for all configured cities",
		})
		return
	}

	// If parameters were provided, use them
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
	if err := c.ShouldBindJSON(&req); err != nil || req.Place == "" {
		// If no parameters provided or invalid JSON, use configured cities
		cities, err := config.GetCityNames(h.db)
		if err != nil {
			h.logger.WithError(err).Error("Failed to get configured cities")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get configured cities"})
			return
		}

		// Start spider for each configured city
		for _, city := range cities {
			normalizedCity := config.NormalizeCity(city)
			err := h.spiderManager.RunSoldSpider(normalizedCity, nil, req.Resume)
			if err != nil {
				h.logger.WithError(err).WithField("city", city).Error("Failed to run sold spider")
				// Continue with other cities even if one fails
				continue
			}
			h.logger.WithField("city", city).Info("Started sold spider successfully")
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Sold spiders started for all configured cities",
		})
		return
	}

	// If parameters were provided, use them
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
	var req models.TelegramConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get existing config
	config, err := h.db.GetTelegramConfig()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get existing config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get existing configuration"})
		return
	}

	// Update the configuration
	if err := h.db.UpdateTelegramConfig(&req); err != nil {
		h.logger.WithError(err).Error("Failed to update config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration"})
		return
	}

	// Update the service configuration
	if config == nil {
		config = &models.TelegramConfig{
			IsEnabled: req.IsEnabled,
			BotToken:  req.BotToken,
			ChatID:    req.ChatID,
		}
	} else {
		config.IsEnabled = req.IsEnabled
		config.BotToken = req.BotToken
		config.ChatID = req.ChatID
	}
	h.telegramService.UpdateConfig(config)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"config": config,
	})
}

// GetTelegramFilters returns the current notification filters
func (h *Handler) GetTelegramFilters(c *gin.Context) {
	filters, err := h.db.GetTelegramFilters()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get Telegram filters")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Telegram filters"})
		return
	}

	c.JSON(http.StatusOK, filters)
}

// UpdateTelegramFilters updates the notification filters
func (h *Handler) UpdateTelegramFilters(c *gin.Context) {
	var filters models.TelegramFilters
	if err := c.ShouldBindJSON(&filters); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate numeric ranges
	if filters.MinPrice != nil && filters.MaxPrice != nil && *filters.MinPrice > *filters.MaxPrice {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Minimum price cannot be greater than maximum price"})
		return
	}
	if filters.MinLivingArea != nil && filters.MaxLivingArea != nil && *filters.MinLivingArea > *filters.MaxLivingArea {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Minimum living area cannot be greater than maximum living area"})
		return
	}
	if filters.MinRooms != nil && filters.MaxRooms != nil && *filters.MinRooms > *filters.MaxRooms {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Minimum rooms cannot be greater than maximum rooms"})
		return
	}

	// Validate districts format (4 digits)
	for _, district := range filters.Districts {
		if len(district) != 4 || !regexp.MustCompile(`^\d{4}$`).MatchString(district) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid district format. Must be 4 digits"})
			return
		}
	}

	// Validate energy labels
	validLabels := map[string]bool{"A++": true, "A+": true, "A": true, "B": true, "C": true, "D": true, "E": true, "F": true, "G": true}
	for _, label := range filters.EnergyLabels {
		if !validLabels[label] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid energy label"})
			return
		}
	}

	if err := h.db.UpdateTelegramFilters(&filters); err != nil {
		h.logger.WithError(err).Error("Failed to update Telegram filters")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save filters"})
		return
	}

	// Update the service's filters
	h.telegramService.UpdateFilters(&filters)

	c.JSON(http.StatusOK, gin.H{"message": "Telegram filters updated successfully"})
}

// TestTelegramConfig tests the Telegram configuration by sending a sample property notification
func (h *Handler) TestTelegramConfig(c *gin.Context) {
	// Get the current configuration from the database
	config, err := h.db.GetTelegramConfig()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get Telegram config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Telegram configuration"})
		return
	}

	if config == nil || !config.IsEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Telegram is not configured or is disabled"})
		return
	}

	// Create a sample property for testing
	sampleProperty := map[string]interface{}{
		"id":              int64(1),
		"street":          "Test Street 123",
		"city":            "Amsterdam",
		"postal_code":     "1012 AB",
		"price":           float64(450000),
		"year_built":      float64(2020),
		"living_area":     float64(85),
		"num_rooms":       float64(3),
		"url":             "https://example.com/test-property",
		"status":          "republished",
		"republish_count": float64(2),
		"energy_label":    "A++",
	}

	// Create a mock district analysis service that doesn't use the database
	mockService := telegram.NewService(h.logger)
	mockService.UpdateConfig(config)

	// Get current filters and apply them to the mock service
	if filters, err := h.db.GetTelegramFilters(); err == nil {
		mockService.UpdateFilters(filters)
	}

	// Set the mock price analysis
	sampleProperty["price_analysis"] = fmt.Sprintf("📊 <u>District Analysis</u>\n" +
		"Current listings (15 properties):\n<b>GOOD</b> (-8.5%% vs. median)\n\n" +
		"Past year sales (42 properties):\n<b>NORMAL</b> (+2.1%% vs. median)")

	// Send test notification
	if err := mockService.NotifyNewProperty(sampleProperty); err != nil {
		h.logger.WithError(err).Error("Failed to send test notification")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test notification sent successfully"})
}

// CheckInitialSetup checks if the database needs initial configuration
func (h *Handler) CheckInitialSetup(c *gin.Context) {
	areas, err := h.db.GetMetropolitanAreas()
	if err != nil {
		h.logger.WithError(err).Error("Failed to check metropolitan areas")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check database state"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"needs_setup": len(areas) == 0,
		"message":     "Database initialization check completed",
	})
}
