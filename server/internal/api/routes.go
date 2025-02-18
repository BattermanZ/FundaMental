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
		api.GET("/properties/stats", handler.GetPropertyStats)
		api.GET("/properties/recent", handler.GetRecentSales)
		api.GET("/properties/area/:postal_prefix", handler.GetAreaStats)
		api.POST("/geocode/update", handler.UpdateCoordinates)
		api.POST("/districts/update", handler.UpdateDistrictHulls)
		api.POST("/spiders/active", handler.RunActiveSpider)
		api.POST("/spiders/sold", handler.RunSoldSpider)

		// Telegram configuration routes
		api.GET("/telegram/config", handler.GetTelegramConfig)
		api.POST("/telegram/config", handler.UpdateTelegramConfig)
	}
}
