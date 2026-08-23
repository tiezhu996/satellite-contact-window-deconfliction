package router

import (
	"github.com/gin-gonic/gin"

	"satellite-contact-window-deconfliction/backend/internal/constants"
	"satellite-contact-window-deconfliction/backend/internal/handler"
	"satellite-contact-window-deconfliction/backend/internal/middleware"
)

func registerSatelliteRoutes(api *gin.RouterGroup, assetHandler *handler.SatelliteAssetHandler) {
	api.GET("/satellites", assetHandler.List)
	api.GET("/satellites/:id", assetHandler.Get)
	planning := api.Group("/satellites", middleware.RBAC(constants.RoleScheduler))
	planning.POST("", assetHandler.Create)
	reviewerUpdate := api.Group("/satellites", middleware.RBAC(constants.RoleReviewer, constants.RoleAdmin))
	reviewerUpdate.PUT("/:id", assetHandler.Update)
}
