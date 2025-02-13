package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Discord struct {
		Token    string   `yaml:"token"`
		GuildID  string   `yaml:"guild_id"`
		Status   string   `yaml:"status"`
		ClientID string   `yaml:"client_id"`
		Devs     []string `yaml:"developers"`
		Sharding struct {
			Enabled     bool `yaml:"enabled"`
			TotalShards int  `yaml:"total_shards"`
		} `yaml:"sharding"`
	} `yaml:"discord"`

	Server struct {
		Port     int    `yaml:"port"`
		Mode     string `yaml:"mode"`
		Host     string `yaml:"host"`
		BasePath string `yaml:"base_path"`
	} `yaml:"server"`

	Riot struct {
		APIKey string `yaml:"api_key"`
		Region string `yaml:"default_region"`
	} `yaml:"riot"`

	Groq struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"groq"`

	MongoDB struct {
		URI      string `yaml:"uri"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mongodb"`

	Lavalink struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		Secure   bool   `yaml:"secure"`
		Name     string `yaml:"name"`
	} `yaml:"lavalink"`

	Logger struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"logger"`

	Cloudinary struct {
		CloudName string `yaml:"cloud_name"`
		APIKey    string `yaml:"api_key"`
		APISecret string `yaml:"api_secret"`
	} `yaml:"cloudinary"`

	ConvertAPI struct {
		Secret string `yaml:"secret"`
	} `yaml:"convertapi"`

	Pterodactyl struct {
		URL    string `yaml:"url"`
		APIKey string `yaml:"api_token"`
	} `yaml:"pterodactyl"`

	Debug        bool      `yaml:"debug"`
	BotStartTime time.Time `yaml:"-"`
}

func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 80
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "release"
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}

	cfg.BotStartTime = time.Now()

	return &cfg, nil
}
