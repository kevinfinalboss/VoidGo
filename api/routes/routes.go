// api/routes/routes.go
package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/controllers"
	"github.com/kevinfinalboss/Void/config"
)

func SetupRoutes(r gin.IRouter, cfg *config.Config) {
	// Controladores
	riotController := controllers.NewRiotController(cfg.Riot.APIKey)
	healthController := controllers.NewHealthController()

	// Rotas públicas
	r.GET("/riot.txt", riotController.ServeRiotTxt)
	r.GET("/health", healthController.CheckHealth)

	// Servir arquivos estáticos
	r.Static("/assets/champions/icons", "./assets/champions/icons")

	// Rotas da API da Riot
	riot := r.Group("/riot")
	{
		riot.GET("/rotation", riotController.GetChampionRotation)
	}
}
