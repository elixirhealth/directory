package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	sq "github.com/Masterminds/squirrel"
	errors2 "github.com/drausin/libri/libri/common/errors"
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/elxirhealth/directory/pkg/server/storage/id"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const (
	pqUniqueViolationErrCode = "23505"
	minSearchQueryLen        = 4
	maxSearchQueryLen        = 32
	minSearchLimit           = 1
	maxSearchLimit           = 8
)

var (
	psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	ErrSearchQueryTooShort = fmt.Errorf("search query shorter than min length %d",
		minSearchQueryLen)
	ErrSearchQueryTooLong = fmt.Errorf("search query longer than max length %d",
		maxSearchQueryLen)
	ErrSearchLimitTooSmall = fmt.Errorf("search limit smaller than min length %d",
		minSearchLimit)
	ErrSearchLimitTooLarge = fmt.Errorf("search limit larger than max length %d",
		maxSearchLimit)

	errEmptyDBUrl            = errors.New("empty DB URL")
	errUnexpectedStorageType = errors.New("unexpected storage type")
)

type postgresStorer struct {
	params  *storage.Parameters
	idGen   id.Generator
	db      *sql.DB
	dbCache sq.DBProxyContext
	qr      querier
	srm     searchResultMerger
}

// New creates a new Storer backed by a Postgres DB at the given dbURL and with the
// given ChecksumIDGenerator.
func New(dbURL string, idGen id.Generator, params *storage.Parameters) (storage.Storer, error) {
	if dbURL == "" {
		return nil, errEmptyDBUrl
	}
	if params.Type != storage.Postgres {
		return nil, errUnexpectedStorageType
	}
	db, err := sql.Open("postgres", dbURL)
	errors2.MaybePanic(err)
	return &postgresStorer{
		params:  params,
		idGen:   idGen,
		db:      db,
		dbCache: sq.NewStmtCacher(db),
		qr:      &querierImpl{},
		srm:     newSearchResultMerger(),
	}, nil
}

func (ps *postgresStorer) PutEntity(e *api.Entity) (string, error) {
	if e.EntityId != "" {
		if err := ps.idGen.Check(e.EntityId); err != nil {
			return "", err
		}
	}
	if err := api.ValidateEntity(e); err != nil {
		return "", err
	}
	insert, err := ps.maybeAddEntityID(e)
	if err != nil {
		return "", err
	}
	tx, err := ps.db.Begin()
	if err != nil {
		return "", err
	}
	fqTbl := fullTableName(storage.GetEntityType(e))
	vals := getPutStmtValues(e)
	ctx, cancel := context.WithTimeout(context.Background(), ps.params.PutQueryTimeout)
	if insert {
		q := psql.RunWith(tx).Insert(fqTbl).SetMap(vals)
		_, err = ps.qr.InsertExecContext(ctx, q)
	} else {
		q := psql.RunWith(tx).Update(fqTbl).SetMap(vals)
		_, err = ps.qr.UpdateExecContext(ctx, q)
	}
	cancel()
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == pqUniqueViolationErrCode {
				return "", storage.ErrDupGenEntityID
			}
		}
		_ = tx.Rollback()
		return "", err
	}
	return e.EntityId, tx.Commit()
}

func (ps *postgresStorer) GetEntity(entityID string) (*api.Entity, error) {
	if err := ps.idGen.Check(entityID); err != nil {
		return nil, err
	}
	et := storage.GetEntityTypeFromID(entityID)
	cols, dest, create := prepEntityScan(et, 0)
	q := psql.RunWith(ps.dbCache).
		Select(cols...).
		From(fullTableName(et)).
		Where(sq.Eq{entityIDCol: entityID})
	ctx, cancel := context.WithTimeout(context.Background(), ps.params.GetQueryTimeout)
	defer cancel()
	row := ps.qr.SelectQueryRowContext(ctx, q)
	if err := row.Scan(dest...); err == sql.ErrNoRows {
		return nil, storage.ErrMissingEntity
	} else if err != nil {
		return nil, err
	}
	return create(), nil
}

func (ps *postgresStorer) SearchEntity(query string, limit uint) ([]*api.Entity, error) {
	if err := ps.validateSearchQuery(query, limit); err != nil {
		return nil, err
	}
	errs := make(chan error, len(searchers))
	wg1 := new(sync.WaitGroup)
	for _, s1 := range searchers {
		wg1.Add(1)
		go func(s2 searcher, wg2 *sync.WaitGroup) {
			defer wg2.Done()
			entityCols, _, _ := prepEntityScan(s2.entityType(), 0)
			selectCols := append(entityCols, s2.similarity())
			q := psql.RunWith(ps.dbCache).
				Select(selectCols...).
				From(fullTableName(s2.entityType())).
				Where(s2.predicate(), s2.preprocQuery(query)).
				OrderBy(similarityCol + " DESC").
				Limit(uint64(limit))
			ctx, cancel := context.WithTimeout(context.Background(),
				ps.params.SearchQueryTimeout)
			defer cancel()
			rows, err := ps.qr.SelectQueryContext(ctx, q)
			if err != nil {
				if err != context.DeadlineExceeded && err != sql.ErrNoRows {
					errs <- err
				}
				return
			}
			if err := ps.srm.merge(rows, s2.name(), s2.entityType()); err != nil {
				errs <- err
				return
			}
			if err := rows.Err(); err != nil {
				errs <- err
				return
			}
			if err := rows.Close(); err != nil {
				errs <- err
				return
			}
		}(s1, wg1)
	}
	wg1.Wait()
	select {
	case err := <-errs:
		return nil, err
	default:
	}

	// return just the entities, without their granular or norm'd similarity scores
	es := make([]*api.Entity, limit)
	for i, eSim := range ps.srm.top(limit) {
		es[i] = eSim.E
	}
	return es, nil
}

func (ps *postgresStorer) Close() error {
	return ps.db.Close()
}

func (ps *postgresStorer) maybeAddEntityID(e *api.Entity) (added bool, err error) {
	if e.EntityId != "" {
		return false, nil
	}
	idPrefix := storage.GetEntityType(e).IDPrefix()
	entityID, err := ps.idGen.Generate(idPrefix)
	if err != nil {
		return false, err
	}
	e.EntityId = entityID
	return true, nil
}

func (ps *postgresStorer) validateSearchQuery(query string, limit uint) error {
	if len(query) < minSearchQueryLen {
		return ErrSearchQueryTooShort
	}
	if len(query) > maxSearchQueryLen {
		return ErrSearchQueryTooLong
	}
	if limit > maxSearchLimit {
		return ErrSearchLimitTooLarge
	}
	if limit < minSearchLimit {
		return ErrSearchLimitTooSmall
	}
	return nil
}

type rows interface {
	Scan(dest ...interface{}) error
	Next() bool
	Close() error
	Err() error
}

type querier interface {
	SelectQueryContext(ctx context.Context, b sq.SelectBuilder) (rows, error)
	SelectQueryRowContext(ctx context.Context, b sq.SelectBuilder) sq.RowScanner
	InsertExecContext(ctx context.Context, b sq.InsertBuilder) (sql.Result, error)
	UpdateExecContext(ctx context.Context, b sq.UpdateBuilder) (sql.Result, error)
}

type querierImpl struct {
}

func (q *querierImpl) SelectQueryContext(
	ctx context.Context, b sq.SelectBuilder,
) (rows, error) {
	return b.QueryContext(ctx)
}

func (q *querierImpl) SelectQueryRowContext(
	ctx context.Context, b sq.SelectBuilder,
) sq.RowScanner {
	return b.QueryRowContext(ctx)
}

func (q *querierImpl) InsertExecContext(
	ctx context.Context, b sq.InsertBuilder,
) (sql.Result, error) {
	return b.ExecContext(ctx)
}

func (q *querierImpl) UpdateExecContext(
	ctx context.Context, b sq.UpdateBuilder,
) (sql.Result, error) {
	return b.ExecContext(ctx)
}
