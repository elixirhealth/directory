package server

import (
	"errors"

	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/elxirhealth/directory/pkg/server/storage/id"
	memstorage "github.com/elxirhealth/directory/pkg/server/storage/memory"
	pgstorage "github.com/elxirhealth/directory/pkg/server/storage/postgres"
	"go.uber.org/zap"
)

var (
	// ErrInvalidStorageType indicates when a storage type is not expected.
	ErrInvalidStorageType = errors.New("invalid storage type")
)

func getStorer(config *Config, logger *zap.Logger) (storage.Storer, error) {
	idGen := id.NewDefaultGenerator()
	switch config.Storage.Type {
	case storage.Memory:
		return memstorage.New(idGen, config.Storage, logger), nil
	case storage.Postgres:
		return pgstorage.New(config.DBUrl, idGen, config.Storage, logger)
	default:
		return nil, ErrInvalidStorageType
	}
}
