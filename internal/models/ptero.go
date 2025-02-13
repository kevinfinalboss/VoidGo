package models

type ServerAttributes struct {
	Identifier     string  `json:"identifier"`
	Name           string  `json:"name"`
	Node           string  `json:"node"`
	State          *string `json:"status"`
	Memory         int     `json:"limits.memory"`
	CPU            int     `json:"limits.cpu"`
	Disk           int     `json:"limits.disk"`
	IsSuspended    bool    `json:"is_suspended"`
	IsInstalling   bool    `json:"is_installing"`
	IsTransferring bool    `json:"is_transferring"`
}

type Server struct {
	Attributes ServerAttributes `json:"attributes"`
}

type ServerListResponse struct {
	Object string   `json:"object"`
	Data   []Server `json:"data"`
}
