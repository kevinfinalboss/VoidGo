package ptero

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/kevinfinalboss/Void/config"
)

type PteroClient struct {
	APIKey  string
	BaseURL string
}

type PowerCommand struct {
	Signal string `json:"signal"`
}

func NewPteroClient(cfg *config.Config) *PteroClient {
	return &PteroClient{
		APIKey:  cfg.Pterodactyl.APIKey,
		BaseURL: cfg.Pterodactyl.URL,
	}
}

func (client *PteroClient) sendRequest(method, endpoint string, payload interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s/api/client/%s", client.BaseURL, endpoint)

	var reqBody []byte
	var err error
	if payload != nil {
		reqBody, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+client.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	clientHttp := &http.Client{}
	resp, err := clientHttp.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Obtém a lista de servidores
func (client *PteroClient) ListServers() ([]byte, error) {
	return client.sendRequest("GET", "", nil)
}

// Envia um comando de energia para um servidor
func (client *PteroClient) SendPowerAction(serverID, action string) error {
	payload := PowerCommand{Signal: action}
	_, err := client.sendRequest("POST", fmt.Sprintf("servers/%s/power", serverID), payload)
	return err
}
