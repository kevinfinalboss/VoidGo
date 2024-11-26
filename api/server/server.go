package server

import (
	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/routes"
)

type Server struct {
	router *gin.Engine
}

func NewServer() *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	return &Server{
		router: router,
	}
}

func (s *Server) SetupRoutes() {
	routes.SetupRoutes(s.router)
}

func (s *Server) Start(addr string) error {
	return s.router.Run(addr)
}
