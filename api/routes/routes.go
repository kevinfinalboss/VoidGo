package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/controllers"
)

func SetupRoutes(router *gin.Engine) {
	riotController := controllers.NewRiotController()
	healthController := controllers.NewHealthController()

	router.GET("/riot.txt", riotController.ServeRiotTxt)
	router.GET("/health", healthController.CheckHealth)
}
