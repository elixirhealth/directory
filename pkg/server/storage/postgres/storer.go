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
	"go.uber.org/zap"
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

	// ErrSearchQueryTooShort identifies when a search query string is shorter than the minimum
	// length.
	ErrSearchQueryTooShort = fmt.Errorf("search query shorter than min length %d",
		minSearchQueryLen)

	// ErrSearchQueryTooLong identifies when a search query string is longer than the maximum
	// length.
	ErrSearchQueryTooLong = fmt.Errorf("search query longer than max length %d",
		maxSearchQueryLen)

	// ErrSearchLimitTooSmall identifies when a search limit is smaller than the minimum value.
	ErrSearchLimitTooSmall = fmt.Errorf("search limit smaller than min length %d",
		minSearchLimit)

	// ErrSearchLimitTooLarge identifies when a search limit is alarger than the maximum value.
	ErrSearchLimitTooLarge = fmt.Errorf("search limit larger than max length %d",
		maxSearchLimit)

	errEmptyDBUrl            = errors.New("empty DB URL")
	errUnexpectedStorageType = errors.New("unexpected storage type")
)

type storer struct {
	params  *storage.Parameters
	idGen   id.Generator
	db      *sql.DB
	dbCache sq.DBProxyContext
	qr      querier
	newSRM  func() searchResultMerger
	logger  *zap.Logger
}

// New creates a new Storer backed by a Postgres DB at the given dbURL and with the
// given ChecksumIDGenerator.
func New(
	dbURL string, idGen id.Generator, params *storage.Parameters, logger *zap.Logger,
) (storage.Storer, error) {
	if dbURL == "" {
		return nil, errEmptyDBUrl
	}
	if params.Type != storage.Postgres {
		return nil, errUnexpectedStorageType
	}
	db, err := sql.Open("postgres", dbURL)
	errors2.MaybePanic(err)
	return &storer{
		params:  params,
		idGen:   idGen,
		db:      db,
		dbCache: sq.NewStmtCacher(db),
		qr:      &querierImpl{},
		newSRM:  func() searchResultMerger { return newSearchResultMerger() },
		logger:  logger,
	}, nil
}

func (ps *storer) PutEntity(e *api.Entity) (string, error) {
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
		q := psql.RunWith(tx).
			Insert(fqTbl).
			SetMap(vals)
		ps.logger.Debug("inserting entity", logPutInsert(q, e)...)
		_, err = ps.qr.InsertExecContext(ctx, q)
	} else {
		q := psql.RunWith(tx).
			Update(fqTbl).
			SetMap(vals).
			Where(sq.Eq{entityIDCol: e.EntityId})
		ps.logger.Debug("updating entity", logPutUpdate(q, e)...)
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
	ps.logger.Debug("successfully stored entity", logPutResult(e.EntityId, insert)...)
	return e.EntityId, tx.Commit()
}

func (ps *storer) GetEntity(entityID string) (*api.Entity, error) {
	if err := ps.idGen.Check(entityID); err != nil {
		return nil, err
	}
	et := storage.GetEntityTypeFromID(entityID)
	cols, dest, create := prepEntityScan(et, 0)
	q := psql.RunWith(ps.dbCache).
		Select(cols...).
		From(fullTableName(et)).
		Where(sq.Eq{entityIDCol: entityID})
	ps.logger.Debug("getting entity", logGetSelect(q, et, entityID)...)
	ctx, cancel := context.WithTimeout(context.Background(), ps.params.GetQueryTimeout)
	defer cancel()
	row := ps.qr.SelectQueryRowContext(ctx, q)
	if err := row.Scan(dest...); err == sql.ErrNoRows {
		return nil, storage.ErrMissingEntity
	} else if err != nil {
		return nil, err
	}
	ps.logger.Debug("successfully found entity", zap.String(logEntityID, entityID))
	return create(), nil
}

func (ps *storer) SearchEntity(query string, limit uint) ([]*api.Entity, error) {
	if err := ps.validateSearchQuery(query, limit); err != nil {
		return nil, err
	}
	errs := make(chan error, len(searchers))
	wg1 := new(sync.WaitGroup)
	srm := ps.newSRM()
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
			ps.logger.Debug("searching for entity", logSearchSelect(q, s2, query)...)
			ctx, cancel := context.WithTimeout(context.Background(),
				ps.params.SearchQueryTimeout)
			defer cancel()
			rows, err := ps.qr.SelectQueryContext(ctx, q)
			n, err := ps.processSearchQuery(srm, rows, err, s2)
			if err != nil {
				errs <- err
			}
			ps.logger.Debug("searcher finished", logSearcherFinished(s2, query, n)...)

		}(s1, wg1)
	}
	wg1.Wait()
	select {
	case err := <-errs:
		return nil, err
	default:
	}

	// return just the entities, without their granular or norm'd similarity scores
	es := make([]*api.Entity, 0, limit)
	ess := srm.top(limit)
	ps.logger.Debug("ranked search results", logSearchRanked(query, limit, ess)...)
	for _, eSim := range ess {
		es = append(es, eSim.E)
	}
	return es, nil
}

func (ps *storer) processSearchQuery(
	srm searchResultMerger, rows queryRows, err error, s searcher,
) (int, error) {
	if err != nil {
		if err != context.DeadlineExceeded && err != sql.ErrNoRows {
			return 0, err
		}
		return 0, nil
	}
	n, err := srm.merge(rows, s.name(), s.entityType())
	if err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return n, err
	}
	if err := rows.Close(); err != nil {
		return n, err
	}
	return n, nil
}

func (ps *storer) Close() error {
	return ps.db.Close()
}

func (ps *storer) maybeAddEntityID(e *api.Entity) (added bool, err error) {
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

func (ps *storer) validateSearchQuery(query string, limit uint) error {
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

type queryRows interface {
	Scan(dest ...interface{}) error
	Next() bool
	Close() error
	Err() error
}

type querier interface {
	SelectQueryContext(ctx context.Context, b sq.SelectBuilder) (queryRows, error)
	SelectQueryRowContext(ctx context.Context, b sq.SelectBuilder) sq.RowScanner
	InsertExecContext(ctx context.Context, b sq.InsertBuilder) (sql.Result, error)
	UpdateExecContext(ctx context.Context, b sq.UpdateBuilder) (sql.Result, error)
}

type querierImpl struct {
}

func (q *querierImpl) SelectQueryContext(
	ctx context.Context, b sq.SelectBuilder,
) (queryRows, error) {
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
