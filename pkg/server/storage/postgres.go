package storage

import (
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/pkg/errors"
)

var (
	errUnknownDBOperation = errors.New("unknown DB operation")
	psql                  = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
)

type postgresStorer struct {
	idGen   ChecksumIDGenerator
	db      *sql.DB
	dbProxy sq.DBProxyBeginner
}

func NewPostgresStorer(dbURL string, idGen ChecksumIDGenerator) (Storer, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	return &postgresStorer{
		idGen:   idGen,
		db:      db,
		dbProxy: sq.NewStmtCacheProxy(db),
	}, nil
}

func (s *postgresStorer) PutEntity(e *api.Entity) (string, error) {
	if err := validateEntity(e); err != nil {
		return "", err
	}
	insert, err := s.maybeAddEntityID(e)
	if err != nil {
		return "", err
	}
	et := getEntityType(e)
	fqTbl := fmt.Sprintf("%s.%s", entitySchema, entityTypeTbls[et])

	tx, err := s.dbProxy.Begin()
	if err != nil {
		return "", err
	}
	if insert {
		_, err = psql.Insert(fqTbl).SetMap(toStmtValues(e)).RunWith(tx).Exec()
	} else {
		_, err = psql.Update(fqTbl).SetMap(toStmtValues(e)).RunWith(tx).Exec()
	}
	if err != nil {
		tx.Rollback()
		return "", err
	}
	return e.EntityId, tx.Commit()
}

func (s *postgresStorer) GetEntity(entityID string) (*api.Entity, error) {
	// TODO (drausin) validate entityID
	et := getEntityTypeFromID(entityID)
	fqTbl := fmt.Sprintf("%s.%s", entitySchema, entityTypeTbls[et])
	conditions := sq.Eq{entityIDCol: entityID}
	row := psql.Select(entityTypeAttrCols[et]...).
		From(fqTbl).
		Where(conditions).
		RunWith(s.db).
		QueryRow()
	return fromRow(row, entityID, et)
}

func (s *postgresStorer) Close() error {
	return s.db.Close()
}

func (s *postgresStorer) maybeAddEntityID(e *api.Entity) (added bool, err error) {
	if e.EntityId != "" {
		return false, nil
	}
	idPrefix := entityTypeIDPrefixes[getEntityType(e)]
	entityID, err := s.idGen.Generate(idPrefix)
	if err != nil {
		return false, err
	}
	e.EntityId = entityID
	return true, nil
}

type entityDBOp uint

const (
	INSERT entityDBOp = iota
	UPDATE
	SELECT
)

type entityType uint

const (
	patient entityType = iota
	office
)

var (
	entitySchema   = "entity"
	entityIDCol    = "entity_id"
	entityTypeTbls = map[entityType]string{
		patient: "patient",
		office:  "office",
	}
	entityTypeAttrCols = map[entityType][]string{
		patient: {
			"last_name",
			"first_name",
			"middle_name",
			"suffix",
			"birthdate",
		},
		office: {
			"name",
		},
	}
	entityTypeIDPrefixes = map[entityType]string{
		patient: "P",
		office:  "F",
	}
)

func getEntityType(e *api.Entity) entityType {
	switch e.TypeAttributes.(type) {
	case *api.Entity_Patient:
		return patient
	case *api.Entity_Office:
		return office
	default:
		panic(errUnknownEntityType)
	}
}

func getEntityTypeFromID(entityID string) entityType {
	for et, prefix := range entityTypeIDPrefixes {
		if strings.HasPrefix(entityID, prefix) {
			return et
		}
	}
	panic(errUnknownEntityType)
}

func toStmtValues(e *api.Entity) map[string]interface{} {
	var vals map[string]interface{}
	switch ta := e.TypeAttributes.(type) {
	case *api.Entity_Patient:
		vals = toPatientStmtValues(ta.Patient)
		vals[entityIDCol] = e.EntityId
	case *api.Entity_Office:
		vals = toOfficeStmtArgs(ta.Office)
		vals[entityIDCol] = e.EntityId
	default:
		panic(errUnknownEntityType)
	}
	return vals
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

func fromPatientRow(row sq.RowScanner, entityID string) (*api.Entity, error) {
	p := &api.Patient{}
	var birthdateISO string
	dest := []interface{}{
		&p.LastName,
		&p.FirstName,
		&p.MiddleName,
		&p.Suffix,
		&birthdateISO,
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	birthdate, err := api.FromISO8601(birthdateISO)
	if err != nil {
		return nil, err
	}
	p.Birthdate = birthdate
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

func fromOfficeRow(row sq.RowScanner, entityID string) (*api.Entity, error) {
	f := &api.Office{}
	dest := []interface{}{
		&f.Name,
	}
	if err := row.Scan(dest); err != nil {
		return nil, err
	}
	return &api.Entity{
		EntityId:       entityID,
		TypeAttributes: &api.Entity_Office{Office: f},
	}, nil
}

func toOfficeStmtArgs(f *api.Office) map[string]interface{} {
	return map[string]interface{}{
		"name": f.Name,
	}
}
