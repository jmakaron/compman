package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmakaron/compman/internal/app/compman/store"
	"github.com/jmakaron/compman/internal/app/compman/types"
)

type pgStore struct {
	ctx    context.Context
	cfg    PGConfig
	cancel context.CancelFunc
	p      *pgxpool.Pool
}

type PGConfig struct {
	Addr     string `json:"addr"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
}

func New(config PGConfig) store.Store {
	return &pgStore{cfg: config}
}

func (s *pgStore) Connect(ctx context.Context) error {
	var err error
	var c *pgxpool.Config
	connUrl := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		s.cfg.Username, s.cfg.Password, s.cfg.Addr, s.cfg.Port, s.cfg.DBName)
	c, err = pgxpool.ParseConfig(connUrl)
	if err != nil {
		return err
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.p, err = pgxpool.NewWithConfig(s.ctx, c)
	if err != nil {
		s.cancel()
		return err
	}
	return nil
}

func (s *pgStore) Disconnect() {
	s.cancel()
	s.p.Close()
}

func (s *pgStore) NewEntity(v interface{}) (store.Entity, error) {
	var err error
	var e store.Entity
	switch v.(type) {
	case *types.Company, types.Company:
		e = &companyEntity{st: s, buff: strings.Builder{}}
	default:
		err = store.ErrUnsupportedType
	}
	return e, err
}
