package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUnsupportedType = errors.New("unsupported type")
	ErrNotConnected    = errors.New("store not connected")
)

type QueryLogEntry struct {
	QStr  string
	QArgs []interface{}
	Start time.Time
	End   time.Time
}

type Entity interface {
	PrepareInsert(interface{}) error
	Insert(context.Context) error
	PrepareSelect(interface{}) error
	Select(context.Context) error
	PrepareUpdate(interface{}) error
	Update(context.Context) error
	PrepareDelete(interface{}) error
	Delete(context.Context) error
	Value() (interface{}, error)
	QueryLog() []QueryLogEntry
}

type Store interface {
	Connect(context.Context) error
	Disconnect()
	NewEntity(interface{}) (Entity, error)
}
