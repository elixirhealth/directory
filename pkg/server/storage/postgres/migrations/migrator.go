package migrations

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/drausin/libri/libri/common/errors"
	"github.com/elxirhealth/service-base/pkg/server/storage"
	"github.com/mattes/migrate"
	"github.com/mattes/migrate/source/go-bindata"
	"go.uber.org/zap"
)

const migrationPrefix = "[migration] "

// TODO (drausin) move to service-base

// Migrator handles Postgres DB migrations. It is a thin wrapper around *Migrate in mattes/migrate
// package.
type Migrator interface {

	// Up migrates the DB up to the latest state.
	Up() error

	// Down migrates the DB all the way to the empty state.
	Down() error
}

type bindataMigrator struct {
	dbURL  string
	as     *bindata.AssetSource
	logger migrate.Logger
}

// NewBindataMigrator creates a new Migrator from the given go-bindata asset source and using the
// given logger.
func NewBindataMigrator(dbURL string, as *bindata.AssetSource, logger migrate.Logger) Migrator {
	return &bindataMigrator{
		dbURL:  dbURL,
		as:     as,
		logger: logger,
	}
}

// Up migrates the DB up to the latest state.
func (bm *bindataMigrator) Up() error {
	m := bm.newInner()
	op := func() error {
		err := m.Up()
		if err == migrate.ErrNoChange {
			return nil
		}
		return err
	}
	if err := backoff.Retry(op, newShortExpBackoff()); err != nil {
		return err
	}
	err1, err2 := m.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// Down migrates the DB down to the empty state.
func (bm *bindataMigrator) Down() error {
	m := bm.newInner()
	if err := m.Down(); err != nil {
		return err
	}
	err1, err2 := m.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (bm *bindataMigrator) newInner() *migrate.Migrate {
	m, err := storage.NewMigrate(bm.dbURL, bm.as)
	errors.MaybePanic(err) // should never happen
	m.Log = bm.logger
	return m
}

func newShortExpBackoff() backoff.BackOff {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Second * 1
	bo.MaxElapsedTime = 10 * time.Second
	return bo
}

// LogLogger implements migrate.Logger via log.Printf
type LogLogger struct{}

// Printf prints the given format and args.
func (ll *LogLogger) Printf(format string, v ...interface{}) {
	log.Printf(migrationPrefix+format, v...)
}

// Verbose indicates whether the logger is verbose. Fixed to false.
func (ll *LogLogger) Verbose() bool {
	return false
}

// ZapLogger implements migrate.Logger by wrapper a *zap.Logger
type ZapLogger struct {
	*zap.Logger
}

// Printf prints the given format and args as INFO messages.
func (zl *ZapLogger) Printf(format string, v ...interface{}) {
	zl.Info(migrationPrefix + fmt.Sprintf(strings.TrimSpace(format), v...))
}

// Verbose indicates whether the logger is verbose. Fixed to false.
func (zl *ZapLogger) Verbose() bool {
	return false
}
