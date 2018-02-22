package storage

import (
	"context"
	"database/sql"
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
	}, nil
}

func (s *postgresStorer) PutEntity(e *api.Entity) (string, error) {
	if e.EntityId != "" {
		if err := s.idGen.Check(e.EntityId); err != nil {
			return "", err
		}
	}
	if err := api.ValidateEntity(e); err != nil {
		return "", err
	}
	insert, err := s.maybeAddEntityID(e)
	if err != nil {
		return "", err
	}
	tx, err := s.dbProxy.Begin()
	if err != nil {
		return "", err
	}
	fqTbl := getEntityType(e).fullTableName()
	vals := toStmtValues(e)
	ctx, cancel := context.WithTimeout(context.Background(), s.params.PutQueryTimeout)
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

func (s *postgresStorer) GetEntity(entityID string) (*api.Entity, error) {
	if err := s.idGen.Check(entityID); err != nil {
		return nil, err
	}
	et := getEntityTypeFromID(entityID)
	ctx, cancel := context.WithTimeout(context.Background(), s.params.GetQueryTimeout)
	row := psql.Select(fromRowCols(et)...).
		From(et.fullTableName()).
		Where(sq.Eq{entityIDCol: entityID}).
		RunWith(s.db).
		QueryRowContext(ctx)
	e, err := fromRow(row, entityID, et)
	cancel()
	if err == sql.ErrNoRows {
		return nil, ErrMissingEntity
	}
	return e, err
}

func (s *postgresStorer) Close() error {
	return s.db.Close()
}

func (s *postgresStorer) maybeAddEntityID(e *api.Entity) (added bool, err error) {
	if e.EntityId != "" {
		return false, nil
	}
	idPrefix := getEntityType(e).idPrefix()
	entityID, err := s.idGen.Generate(idPrefix)
	if err != nil {
		return false, err
	}
	e.EntityId = entityID
	return true, nil
}

const (
	entitySchema = "entity"
	entityIDCol  = "entity_id"
)

func (et entityType) fullTableName() string {
	return entitySchema + "." + et.string()
}

func toStmtValues(e *api.Entity) map[string]interface{} {
	var vals map[string]interface{}
	switch ta := e.TypeAttributes.(type) {
	case *api.Entity_Patient:
		vals = toPatientStmtValues(ta.Patient)
		vals[entityIDCol] = e.EntityId
	case *api.Entity_Office:
		vals = toOfficeStmtValues(ta.Office)
		vals[entityIDCol] = e.EntityId
	default:
		panic(errUnknownEntityType)
	}
	return vals
}

func fromRowCols(et entityType) []string {
	switch et {
	case patient:
		return fromPatientRowCols
	case office:
		return fromOfficeRowCols
	default:
		panic(errUnknownEntityType)
	}
}

func fromRow(row sq.RowScanner, entityID string, et entityType) (*api.Entity, error) {
	switch et {
	case patient:
		return fromPatientRow(row, entityID)
	case office:
		return fromOfficeRow(row, entityID)
	default:
		panic(errUnknownEntityType)
	}
}

var fromPatientRowCols = []string{
	"last_name",
	"first_name",
	"middle_name",
	"suffix",
	"birthdate",
}

func fromPatientRow(row sq.RowScanner, entityID string) (*api.Entity, error) {
	p := &api.Patient{}
	var birthdateTime time.Time
	dest := []interface{}{
		&p.LastName,
		&p.FirstName,
		&p.MiddleName,
		&p.Suffix,
		&birthdateTime,
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	p.Birthdate = &api.Date{
		Year:  uint32(birthdateTime.Year()),
		Month: uint32(birthdateTime.Month()),
		Day:   uint32(birthdateTime.Day()),
	}
	return &api.Entity{
		EntityId:       entityID,
		TypeAttributes: &api.Entity_Patient{Patient: p},
	}, nil
}

func toPatientStmtValues(p *api.Patient) map[string]interface{} {
	return map[string]interface{}{
		"last_name":   p.LastName,
		"first_name":  p.FirstName,
		"middle_name": p.MiddleName,
		"suffix":      p.Suffix,
		"birthdate":   p.Birthdate.ISO8601(),
	}
}

var fromOfficeRowCols = []string{
	"name",
}

func fromOfficeRow(row sq.RowScanner, entityID string) (*api.Entity, error) {
	f := &api.Office{}
	dest := []interface{}{
		&f.Name,
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	return &api.Entity{
		EntityId:       entityID,
		TypeAttributes: &api.Entity_Office{Office: f},
	}, nil
}

func toOfficeStmtValues(f *api.Office) map[string]interface{} {
	return map[string]interface{}{
		"name": f.Name,
	}
}
