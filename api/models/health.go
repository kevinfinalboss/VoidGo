package models

import "time"

type HealthResponse struct {
	Status    string `json:"status"`
	Uptime    string `json:"uptime"`
	Timestamp string `json:"timestamp"`
}

type ServiceStatus string

const (
	StatusHealthy   ServiceStatus = "healthy"
	StatusUnhealthy ServiceStatus = "unhealthy"
)

func NewHealthResponse(uptime time.Duration) HealthResponse {
	return HealthResponse{
		Status:    string(StatusHealthy),
		Uptime:    uptime.String(),
		Timestamp: time.Now().Format(time.RFC3339),
	}
}
