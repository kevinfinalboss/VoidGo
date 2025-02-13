package models

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

type RotationResponse struct {
	FreeChampions      []string `json:"freeChampions"`
	NewPlayerChampions []string `json:"newPlayerChampions"`
	MaxNewPlayerLevel  int      `json:"maxNewPlayerLevel"`
}
