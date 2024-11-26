// api/server/server.go
package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/routes"
	"github.com/kevinfinalboss/Void/config"
)

type Server struct {
	router *gin.Engine
	config *config.Config
}

func NewServer(cfg *config.Config) *Server {
	// Configura o modo do Gin baseado na configuração
	gin.SetMode(cfg.Server.Mode)

	// Cria uma nova instância do router
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	return &Server{
		router: router,
		config: cfg,
	}
}

func (s *Server) SetupRoutes() {
	if s.config.Server.BasePath != "" {
		group := s.router.Group(s.config.Server.BasePath)
		routes.SetupRoutes(group)
	} else {
		routes.SetupRoutes(s.router)
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	return s.router.Run(addr)
}
