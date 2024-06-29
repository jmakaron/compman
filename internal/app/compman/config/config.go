package config

import (
	"encoding/json"
	"os"

	"github.com/jmakaron/compman/internal/pkg/http"
)

type Config struct {
	HttpCfg http.HTTPServiceCfg `json:"http"`
}

func ParseConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err = json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
