package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type RiotController struct{}

func NewRiotController() *RiotController {
	return &RiotController{}
}

func (rc *RiotController) ServeRiotTxt(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "a66bc314-412d-4e19-9ab0-56008f94b90a")
}
