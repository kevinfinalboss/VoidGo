package server

import (
	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/routes"
)

type Server struct {
	router *gin.Engine
}

func NewServer() *Server {
	return &Server{
		router: gin.Default(),
	}
}

func (s *Server) SetupRoutes() {
	routes.SetupRiotRoutes(s.router)
}

func (s *Server) Start(addr string) error {
	return s.router.Run(addr)
}
