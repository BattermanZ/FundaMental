package api

import (
	"fundamental/server/internal/database"
	"fundamental/server/internal/geocoding"
	"fundamental/server/internal/models"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MetropolitanHandler struct {
	db       *database.Database
	geocoder *geocoding.Geocoder
}

func NewMetropolitanHandler(db *database.Database, geocoder *geocoding.Geocoder) *MetropolitanHandler {
	return &MetropolitanHandler{
		db:       db,
		geocoder: geocoder,
	}
}

// SetupMetropolitanRoutes adds metropolitan area routes to the router
func SetupMetropolitanRoutes(router *gin.Engine, db *database.Database, geocoder *geocoding.Geocoder) {
	handler := NewMetropolitanHandler(db, geocoder)

	router.GET("/api/metropolitan", handler.ListMetropolitanAreas)
	router.POST("/api/metropolitan", handler.CreateMetropolitanArea)
	router.GET("/api/metropolitan/:name", handler.GetMetropolitanArea)
	router.PUT("/api/metropolitan/:name", handler.UpdateMetropolitanArea)
	router.DELETE("/api/metropolitan/:name", handler.DeleteMetropolitanArea)
	router.POST("/api/metropolitan/:name/geocode", handler.GeocodeMetropolitanArea)
}

// ListMetropolitanAreas returns all metropolitan areas
func (h *MetropolitanHandler) ListMetropolitanAreas(c *gin.Context) {
	areas, err := h.db.GetMetropolitanAreas()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, areas)
}

// GetMetropolitanArea returns a specific metropolitan area
func (h *MetropolitanHandler) GetMetropolitanArea(c *gin.Context) {
	name := c.Param("name")
	area, err := h.db.GetMetropolitanAreaByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if area == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Metropolitan area not found"})
		return
	}
	c.JSON(http.StatusOK, area)
}

// CreateMetropolitanArea creates a new metropolitan area
func (h *MetropolitanHandler) CreateMetropolitanArea(c *gin.Context) {
	var area models.MetropolitanArea
	if err := c.ShouldBindJSON(&area); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.UpdateMetropolitanArea(area); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// After creating the area, trigger geocoding
	go h.geocodeArea(&area)

	c.JSON(http.StatusCreated, area)
}

// UpdateMetropolitanArea updates an existing metropolitan area
func (h *MetropolitanHandler) UpdateMetropolitanArea(c *gin.Context) {
	name := c.Param("name")
	var area models.MetropolitanArea
	if err := c.ShouldBindJSON(&area); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure the name in the URL matches the name in the body
	if area.Name != name {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name in URL does not match name in body"})
		return
	}

	if err := h.db.UpdateMetropolitanArea(area); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// After updating the area, trigger geocoding
	go h.geocodeArea(&area)

	c.JSON(http.StatusOK, area)
}

// DeleteMetropolitanArea deletes a metropolitan area
func (h *MetropolitanHandler) DeleteMetropolitanArea(c *gin.Context) {
	name := c.Param("name")
	if err := h.db.DeleteMetropolitanArea(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GeocodeMetropolitanArea handles geocoding of cities in a metropolitan area
func (h *MetropolitanHandler) GeocodeMetropolitanArea(c *gin.Context) {
	name := c.Param("name")

	// Get the metropolitan area
	area, err := h.db.GetMetropolitanAreaByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metropolitan area"})
		return
	}

	if area == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Metropolitan area not found"})
		return
	}

	// Process each city
	for _, city := range area.Cities {
		// Try to geocode the city
		result, err := h.geocoder.GeocodeCity(city)
		if err != nil {
			// Log the error but continue with other cities
			log.Printf("Failed to geocode city %s: %v", city, err)
			continue
		}

		// Update the coordinates in the database
		err = h.db.UpdateCityCoordinates(area.ID, city, result.Lat, result.Lng)
		if err != nil {
			log.Printf("Failed to update coordinates for city %s: %v", city, err)
			continue
		}
	}

	// Get the updated metropolitan area
	updatedArea, err := h.db.GetMetropolitanAreaByName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get updated metropolitan area"})
		return
	}

	// Return the updated area
	c.JSON(http.StatusOK, updatedArea)
}

// geocodeArea is a helper function to geocode all cities in a metropolitan area
func (h *MetropolitanHandler) geocodeArea(area *models.MetropolitanArea) {
	for _, city := range area.Cities {
		result, err := h.geocoder.GeocodeCity(city)
		if err != nil {
			log.Printf("Failed to geocode city %s: %v", city, err)
			continue
		}

		err = h.db.UpdateCityCoordinates(area.ID, city, result.Lat, result.Lng)
		if err != nil {
			log.Printf("Failed to update coordinates for city %s: %v", city, err)
			continue
		}
	}
}
