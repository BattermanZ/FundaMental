package api

import (
	"fundamental/server/internal/database"
	"fundamental/server/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupMetropolitanRoutes adds metropolitan area routes to the router
func SetupMetropolitanRoutes(router *gin.Engine, db *database.Database) {
	metropolitan := router.Group("/api/metropolitan")
	{
		metropolitan.GET("", getMetropolitanAreas(db))
		metropolitan.GET("/:name", getMetropolitanArea(db))
		metropolitan.POST("", createMetropolitanArea(db))
		metropolitan.PUT("/:name", updateMetropolitanArea(db))
		metropolitan.DELETE("/:name", deleteMetropolitanArea(db))
	}
}

// getMetropolitanAreas returns all metropolitan areas
func getMetropolitanAreas(db *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		areas, err := db.GetMetropolitanAreas()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, areas)
	}
}

// getMetropolitanArea returns a specific metropolitan area
func getMetropolitanArea(db *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		area, err := db.GetMetropolitanAreaByName(name)
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
}

// createMetropolitanArea creates a new metropolitan area
func createMetropolitanArea(db *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var area models.MetropolitanArea
		if err := c.ShouldBindJSON(&area); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.UpdateMetropolitanArea(area); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, area)
	}
}

// updateMetropolitanArea updates an existing metropolitan area
func updateMetropolitanArea(db *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		if err := db.UpdateMetropolitanArea(area); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, area)
	}
}

// deleteMetropolitanArea deletes a metropolitan area
func deleteMetropolitanArea(db *database.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := db.DeleteMetropolitanArea(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
