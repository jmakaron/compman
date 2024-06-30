package config

import (
	"encoding/json"
	"os"

	"github.com/jmakaron/compman/internal/app/compman/store/postgres"
	"github.com/jmakaron/compman/internal/pkg/http"
)

type AppConfig struct {
	HttpCfg http.HTTPServiceCfg `json:"http"`
	Db      postgres.PGConfig   `json:"db"`
}

func ParseConfigFile(path string) (*AppConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err = json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
