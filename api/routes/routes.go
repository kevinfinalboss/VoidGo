package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/controllers"
	"github.com/kevinfinalboss/Void/config"
)

func SetupRoutes(r gin.IRouter, cfg *config.Config) {
	riotController := controllers.NewRiotController(cfg.Riot.APIKey)
	healthController := controllers.NewHealthController()

	r.GET("/riot.txt", riotController.ServeRiotTxt)
	r.GET("/health", healthController.CheckHealth)

	r.Static("/assets/champions/icons", "./assets/champions/icons")

	riot := r.Group("/riot")
	{
		riot.GET("/rotation", riotController.GetChampionRotation)
	}
}
