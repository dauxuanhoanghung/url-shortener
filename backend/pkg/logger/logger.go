package logger

import (
	"go.uber.org/zap"
)

func New(mode string) (*zap.Logger, error) {
	if mode == "release" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
