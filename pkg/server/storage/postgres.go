package storage

import (
	"context"
	"database/sql"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	errors2 "github.com/drausin/libri/libri/common/errors"
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const (
	pqUniqueViolationErrCode = "23505"
)

var (
	psql                     = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	errEmptyDBUrl            = errors.New("empty DB URL")
	errUnexpectedStorageType = errors.New("unexpected storage type")
)

type postgresStorer struct {
	params  *Parameters
	idGen   ChecksumIDGenerator
	db      *sql.DB
	dbProxy sq.DBProxyBeginner
	srm     searchResultMerger
}

// NewPostgres creates a new Storer backed by a Postgres DB at the given dbURL and with the
// given ChecksumIDGenerator.
func NewPostgres(dbURL string, idGen ChecksumIDGenerator, params *Parameters) (Storer, error) {
	if dbURL == "" {
		return nil, errEmptyDBUrl
	}
	if params.Type != Postgres {
		return nil, errUnexpectedStorageType
	}
	db, err := sql.Open("postgres", dbURL)
	errors2.MaybePanic(err)
	return &postgresStorer{
		params:  params,
		idGen:   idGen,
		db:      db,
		dbProxy: sq.NewStmtCacheProxy(db),
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
	tx, err := ps.dbProxy.Begin()
	if err != nil {
		return "", err
	}
	fqTbl := getEntityType(e).fullTableName()
	vals := toStmtValues(e)
	ctx, cancel := context.WithTimeout(context.Background(), ps.params.PutQueryTimeout)
	if insert {
		_, err = psql.Insert(fqTbl).SetMap(vals).RunWith(tx).ExecContext(ctx)
	} else {
		_, err = psql.Update(fqTbl).SetMap(vals).RunWith(tx).ExecContext(ctx)
	}
	cancel()
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == pqUniqueViolationErrCode {
				return "", ErrDupGenEntityID
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
	et := getEntityTypeFromID(entityID)
	ctx, cancel := context.WithTimeout(context.Background(), ps.params.GetQueryTimeout)
	defer cancel()
	cols, dest, scan := prepEntityScan(et, 0)
	row := psql.Select(cols...).
		From(et.fullTableName()).
		Where(sq.Eq{entityIDCol: entityID}).
		RunWith(ps.db).
		QueryRowContext(ctx)
	if err := row.Scan(dest...); err == sql.ErrNoRows {
		return nil, ErrMissingEntity
	} else if err != nil {
		return nil, err
	}
	return scan(), nil
}

func (ps *postgresStorer) SearchEntity(query string, limit uint) ([]*api.Entity, error) {
	// TODO (drausin) check query ok (len >= 3) and limit ok (> 0)
	errs := make(chan error, len(searchers))

	wg1 := new(sync.WaitGroup)
	for _, s1 := range searchers {
		wg1.Add(1)
		go func(s2 searcher, wg2 *sync.WaitGroup) {
			defer wg2.Done()
			entityCols, _, _ := prepEntityScan(s2.entityType(), 0)
			selectCols := append(entityCols, s2.similarity())
			q := psql.Select(selectCols...).
				From(s2.entityType().fullTableName()).
				Where(s2.predicate(), s2.preprocQuery(query)).
				OrderBy(similarityCol + " DESC").
				Limit(uint64(limit))
			ctx, cancel := context.WithTimeout(context.Background(),
				ps.params.SearchQueryTimeout)
			rows, err := q.RunWith(ps.db).QueryContext(ctx)
			if err != nil && err != context.DeadlineExceeded {
				errs <- err
				return
			} else if err == context.DeadlineExceeded {
				return
			}
			if err := ps.srm.merge(rows, s2.name(), s2.entityType()); err != nil {
				errs <- err
				return
			}
			cancel()
			if err := rows.Err(); err != nil {
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
		es[i] = eSim.e
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
	idPrefix := getEntityType(e).idPrefix()
	entityID, err := ps.idGen.Generate(idPrefix)
	if err != nil {
		return false, err
	}
	e.EntityId = entityID
	return true, nil
}

const (
	entitySchema  = "entity"
	entityIDCol   = "entity_id"
	similarityCol = "sim"

	// patient attribute indexedValue
	lastNameCol   = "last_name"
	firstNameCol  = "first_name"
	middleNameCol = "middle_name"
	suffixCol     = "suffix"
	birthdateCol  = "birthdate"

	// office attribute indexedValue
	nameCol = "name"
)

func (et entityType) fullTableName() string {
	return entitySchema + "." + et.string()
}

func toStmtValues(e *api.Entity) map[string]interface{} {
	var vals map[string]interface{}
	switch ta := e.TypeAttributes.(type) {
	case *api.Entity_Patient:
		vals = toPatientStmtValues(ta.Patient)
	case *api.Entity_Office:
		vals = toOfficeStmtValues(ta.Office)
	default:
		panic(errUnknownEntityType)
	}
	vals[entityIDCol] = e.EntityId
	return vals
}

func prepEntityScan(
	et entityType, extraDest int,
) (cols []string, dest []interface{}, create func() *api.Entity) {
	switch et {
	case patient:
		return prepPatientScan(extraDest)
	case office:
		return prepOfficeScan(extraDest)
	default:
		panic(errUnknownEntityType)
	}
}

func prepPatientScan(extraDest int) (cols []string, dest []interface{}, create func() *api.Entity) {
	p := &api.Patient{}
	e := &api.Entity{TypeAttributes: &api.Entity_Patient{Patient: p}}
	var birthdateTime time.Time
	colDests := []struct {
		col  string
		dest interface{}
	}{
		{entityIDCol, &e.EntityId},
		{lastNameCol, &p.LastName},
		{firstNameCol, &p.FirstName},
		{middleNameCol, &p.MiddleName},
		{suffixCol, &p.Suffix},
		{birthdateCol, &birthdateTime},
	}
	dest = make([]interface{}, len(colDests), len(colDests)+extraDest)
	cols = make([]string, len(colDests))
	for i, colDest := range colDests {
		cols[i] = colDest.col
		dest[i] = colDest.dest
	}

	return cols, dest, func() *api.Entity {
		e.EntityId = *dest[0].(*string)
		p.LastName = *dest[1].(*string)
		p.FirstName = *dest[2].(*string)
		p.MiddleName = *dest[3].(*string)
		p.Suffix = *dest[4].(*string)
		birthdateTime := *dest[5].(*time.Time)
		p.Birthdate = &api.Date{
			Year:  uint32(birthdateTime.Year()),
			Month: uint32(birthdateTime.Month()),
			Day:   uint32(birthdateTime.Day()),
		}
		return e
	}
}

func prepOfficeScan(extraDest int) (cols []string, dest []interface{}, create func() *api.Entity) {
	f := &api.Office{}
	e := &api.Entity{TypeAttributes: &api.Entity_Office{Office: f}}
	colDests := []struct {
		col  string
		dest interface{}
	}{
		{entityIDCol, &e.EntityId},
		{nameCol, &f.Name},
	}

	dest = make([]interface{}, len(colDests), len(colDests)+extraDest)
	cols = make([]string, len(colDests))
	for i, colDest := range colDests {
		cols[i] = colDest.col
		dest[i] = colDest.dest
	}

	return cols, dest, func() *api.Entity {
		e.EntityId = *dest[0].(*string)
		f.Name = *dest[1].(*string)
		return e
	}
}

func toPatientStmtValues(p *api.Patient) map[string]interface{} {
	return map[string]interface{}{
		lastNameCol:   p.LastName,
		firstNameCol:  p.FirstName,
		middleNameCol: p.MiddleName,
		suffixCol:     p.Suffix,
		birthdateCol:  p.Birthdate.ISO8601(),
	}
}

func toOfficeStmtValues(f *api.Office) map[string]interface{} {
	return map[string]interface{}{
		nameCol: f.Name,
	}
}
