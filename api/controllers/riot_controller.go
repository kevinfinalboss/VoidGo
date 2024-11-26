// api/controllers/riot_controller.go
package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kevinfinalboss/Void/api/models"
	"github.com/kevinfinalboss/Void/api/services"
)

type RiotController struct {
	riotService *services.RiotService
}

func NewRiotController(apiKey string) *RiotController {
	return &RiotController{
		riotService: services.NewRiotService(apiKey),
	}
}

func (rc *RiotController) ServeRiotTxt(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "a66bc314-412d-4e19-9ab0-56008f94b90a")
}

func (rc *RiotController) GetChampionRotation(c *gin.Context) {
	region := c.DefaultQuery("region", "br1")

	rotations, err := rc.riotService.GetChampionRotations(region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	var freeChampions []string
	var newPlayerChampions []string

	for _, id := range rotations.FreeChampionIds {
		name, err := rc.riotService.GetChampionNameById(id)
		if err == nil {
			freeChampions = append(freeChampions, name)
		}
	}

	for _, id := range rotations.FreeChampionIdsForNewPlayers {
		name, err := rc.riotService.GetChampionNameById(id)
		if err == nil {
			newPlayerChampions = append(newPlayerChampions, name)
		}
	}

	response := models.RotationResponse{
		FreeChampions:      freeChampions,
		NewPlayerChampions: newPlayerChampions,
		MaxNewPlayerLevel:  rotations.MaxNewPlayerLevel,
	}

	c.JSON(http.StatusOK, response)
}
