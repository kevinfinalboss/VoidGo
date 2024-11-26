// api/routes/routes.go
package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/controllers"
)

func SetupRoutes(r gin.IRouter) {
	// Controladores
	riotController := controllers.NewRiotController()
	healthController := controllers.NewHealthController()

	// Rotas públicas
	r.GET("/riot.txt", riotController.ServeRiotTxt)
	r.GET("/health", healthController.CheckHealth)
}
