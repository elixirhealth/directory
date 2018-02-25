package server

import (
	"errors"

	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/elxirhealth/directory/pkg/server/storage/id"
	pgstorage "github.com/elxirhealth/directory/pkg/server/storage/postgres"
)

var (
	// ErrInvalidStorageType indicates when a storage type is not expected.
	ErrInvalidStorageType = errors.New("invalid storage type")
)

func getStorer(config *Config) (storage.Storer, error) {
	idGen := id.NewDefaultGenerator()
	switch config.Storage.Type {
	case storage.Postgres:
		return pgstorage.New(config.DBUrl, idGen, config.Storage)
	default:
		return nil, ErrInvalidStorageType
	}
}
