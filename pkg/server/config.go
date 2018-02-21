package server

import (
	"github.com/drausin/libri/libri/common/errors"
	"github.com/elxirhealth/service-base/pkg/server"
	"go.uber.org/zap/zapcore"
)

const (
// TODO add default config values here
)

// Config is the config for a Directory instance.
type Config struct {
	*server.BaseConfig
	// TODO add config elements
}

// NewDefaultConfig create a new config instance with default values.
func NewDefaultConfig() *Config {
	config := &Config{
		BaseConfig: server.NewDefaultBaseConfig(),
	}
	return config // TODO add .WithDefaultCONFIGELEMENT for each CONFIGELEMENT
}

// MarshalLogObject writes the config to the given object encoder.
func (c *Config) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	err := c.BaseConfig.MarshalLogObject(oe)
	errors.MaybePanic(err) // should never happen

	// TODO add other config elements
	return nil
}

// TODO add WithCONFIGELEMENT and WithDefaultCONFIGELEMENT methods for each CONFIGELEMENT
