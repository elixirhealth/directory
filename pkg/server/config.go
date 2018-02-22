package server

import (
	"github.com/drausin/libri/libri/common/errors"
	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/elxirhealth/service-base/pkg/server"
	"go.uber.org/zap/zapcore"
)

// Config is the config for a Directory instance.
type Config struct {
	*server.BaseConfig
	Storage *storage.Parameters
	DBUrl   string
}

// NewDefaultConfig create a new config instance with default values.
func NewDefaultConfig() *Config {
	config := &Config{
		BaseConfig: server.NewDefaultBaseConfig(),
	}
	return config.
		WithDefaultStorage()
}

// MarshalLogObject writes the config to the given object encoder.
func (c *Config) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	err := c.BaseConfig.MarshalLogObject(oe)
	errors.MaybePanic(err) // should never happen
	err = oe.AddObject(logStorage, c.Storage)
	errors.MaybePanic(err) // should never happen
	oe.AddString(logDBUrl, c.DBUrl)
	return nil
}

// WithStorage sets the cache parameters to the given value or the defaults if it is nil.
func (c *Config) WithStorage(p *storage.Parameters) *Config {
	if p == nil {
		return c.WithDefaultStorage()
	}
	c.Storage = p
	return c
}

// WithDefaultStorage set the Cache parameters to their default values.
func (c *Config) WithDefaultStorage() *Config {
	c.Storage = storage.NewDefaultParameters()
	return c
}

// WithDBUrl sets the DB URL to the given value.
func (c *Config) WithDBUrl(dbURL string) *Config {
	c.DBUrl = dbURL
	return c
}
