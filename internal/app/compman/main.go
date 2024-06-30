package compman

import (
	"context"
	"fmt"

	"github.com/jmakaron/compman/internal/app/compman/config"
	"github.com/jmakaron/compman/internal/app/compman/store"
	"github.com/jmakaron/compman/internal/app/compman/store/postgres"
	httpsrv "github.com/jmakaron/compman/internal/pkg/http"
	"github.com/jmakaron/compman/pkg/logger"
)

type ServiceComponent struct {
	log    *logger.Logger
	cfg    *config.AppConfig
	st     store.Store
	ep     *httpsrv.HTTPService
	ctx    context.Context
	cancel context.CancelFunc
}

func New(log *logger.Logger) *ServiceComponent {
	return &ServiceComponent{log: log}
}

func (c *ServiceComponent) Init(cfg *config.AppConfig) error {
	c.cfg = cfg
	c.st = postgres.New(c.cfg.Db)
	c.ep = &httpsrv.HTTPService{}
	return nil
}

func (c *ServiceComponent) Start() error {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	if err := c.st.Connect(c.ctx); err != nil {
		c.log.Debug("could not connect to db")
		return err
	}
	layout, spec := c.getRestAPI()
	if err := c.ep.Init(c.cfg.HttpCfg, layout, spec, c.log); err != nil {
		c.log.Debug("could not initialize http service component")
		c.st.Disconnect()
		return err
	}
	if err := c.ep.Start(); err != nil {
		c.log.Debug("could not start http service component")
		return err
	}
	return nil
}

func (c *ServiceComponent) Stop() {
	if err := c.ep.Stop(); err != nil {
		c.log.Error(fmt.Sprintf("failed to stop http service component, %+v", err))
	}
	c.st.Disconnect()
	c.cancel()
}
