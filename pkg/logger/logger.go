package logger

import "go.uber.org/zap"

type Logger struct {
	*zap.Logger
}

func New(debug bool) (*Logger, error) {
	var zl *zap.Logger
	var err error
	if debug {
		zl, err = zap.NewDevelopment()
	} else {
		zl, err = zap.NewProduction()
	}
	return &Logger{zl}, err
}
