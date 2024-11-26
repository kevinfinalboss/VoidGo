package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/models"
)

type HealthController struct {
	startTime time.Time
}

func NewHealthController() *HealthController {
	return &HealthController{
		startTime: time.Now(),
	}
}

func (hc *HealthController) CheckHealth(c *gin.Context) {
	uptime := time.Since(hc.startTime)
	response := models.NewHealthResponse(uptime)
	c.JSON(http.StatusOK, response)
}
