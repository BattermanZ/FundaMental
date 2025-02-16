package api

import (
	"fundamental/server/internal/database"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, db *database.Database) {
	handler := NewHandler(db, nil)

	api := router.Group("/api")
	{
		api.GET("/properties", handler.GetAllProperties)
		api.GET("/stats", handler.GetPropertyStats)
		api.GET("/areas/:postal_prefix", handler.GetAreaStats)
		api.GET("/recent-sales", handler.GetRecentSales)
		api.POST("/update-coordinates", handler.UpdateCoordinates)
	}
} 