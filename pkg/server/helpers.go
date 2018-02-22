package server

import (
	"errors"

	"github.com/elxirhealth/directory/pkg/server/storage"
)

var (
	// ErrInvalidStorageType indicates when a storage type is not expected.
	ErrInvalidStorageType = errors.New("invalid storage type")
)

func getStorer(config *Config) (storage.Storer, error) {
	idGen := storage.NewDefaultIDGenerator()
	switch config.Storage.Type {
	case storage.Postgres:
		return storage.NewPostgres(config.DBUrl, idGen, config.Storage)
	default:
		return nil, ErrInvalidStorageType
	}
}
