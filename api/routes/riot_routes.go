package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/controllers"
)

func SetupRiotRoutes(router *gin.Engine) {
	riotController := controllers.NewRiotController()
	router.GET("/riot.txt", riotController.ServeRiotTxt)
}
