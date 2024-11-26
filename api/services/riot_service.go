package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ChampionInfo struct {
	FreeChampionIds              []int `json:"freeChampionIds"`
	FreeChampionIdsForNewPlayers []int `json:"freeChampionIdsForNewPlayers"`
	MaxNewPlayerLevel            int   `json:"maxNewPlayerLevel"`
}

type ChampionData struct {
	Type    string                    `json:"type"`
	Format  string                    `json:"format"`
	Version string                    `json:"version"`
	Data    map[string]ChampionDetail `json:"data"`
}

type ChampionDetail struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Name  string `json:"name"`
	Title string `json:"title"`
	Image struct {
		Full string `json:"full"`
	} `json:"image"`
}

type RiotService struct {
	apiKey     string
	httpClient *http.Client
	champions  *ChampionData
	mutex      sync.RWMutex
	lastUpdate time.Time
	version    string
}

func NewRiotService(apiKey string) *RiotService {
	return &RiotService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *RiotService) GetLatestVersion() (string, error) {
	if s.version != "" {
		return s.version, nil
	}

	resp, err := s.httpClient.Get("https://ddragon.leagueoflegends.com/api/versions.json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var versions []string
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no versions available")
	}

	s.version = versions[0]
	return s.version, nil
}

func (s *RiotService) GetChampionRotations(region string) (*ChampionInfo, error) {
	url := fmt.Sprintf("https://%s.api.riotgames.com/lol/platform/v3/champion-rotations", region)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Riot-Token", s.apiKey)
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Charset", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var rotations ChampionInfo
	if err := json.NewDecoder(resp.Body).Decode(&rotations); err != nil {
		return nil, err
	}

	return &rotations, nil
}

func (s *RiotService) GetChampionsData() (*ChampionData, error) {
	s.mutex.RLock()
	if time.Since(s.lastUpdate) < 24*time.Hour && s.champions != nil {
		defer s.mutex.RUnlock()
		return s.champions, nil
	}
	s.mutex.RUnlock()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if time.Since(s.lastUpdate) < 24*time.Hour && s.champions != nil {
		return s.champions, nil
	}

	url := "https://ddragon.leagueoflegends.com/cdn/14.1.1/data/pt_BR/champion.json"
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var champData ChampionData
	if err := json.NewDecoder(resp.Body).Decode(&champData); err != nil {
		return nil, err
	}

	s.champions = &champData
	s.lastUpdate = time.Now()

	return &champData, nil
}

func (s *RiotService) GetChampionNameById(id int) (string, error) {
	champData, err := s.GetChampionsData()
	if err != nil {
		return "", err
	}

	idStr := fmt.Sprintf("%d", id)
	for _, champ := range champData.Data {
		if champ.Key == idStr {
			return champ.Name, nil
		}
	}

	return "", fmt.Errorf("champion not found with id: %d", id)
}
