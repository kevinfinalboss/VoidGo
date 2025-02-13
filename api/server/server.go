package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/routes"
	"github.com/kevinfinalboss/Void/config"
)

type Server struct {
	router     *gin.Engine
	config     *config.Config
	httpServer *http.Server
}

func NewServer(cfg *config.Config) *Server {
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return &Server{
		router:     router,
		config:     cfg,
		httpServer: httpServer,
	}
}

func (s *Server) SetupRoutes() {
	if s.config.Server.BasePath != "" {
		group := s.router.Group(s.config.Server.BasePath)
		routes.SetupRoutes(group, s.config)
	} else {
		routes.SetupRoutes(s.router, s.config)
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
