package storage

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	sq "github.com/Masterminds/squirrel"
	errors2 "github.com/drausin/libri/libri/common/errors"
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
	stmts   map[entityType]map[entityDBOp]*sql.Stmt
}

func NewPostgresStorer(dbURL string, idGen ChecksumIDGenerator) (Storer, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	dbProxy := sq.NewStmtCacheProxy(db)
	stmts := make(map[entityType]map[entityDBOp]*sql.Stmt)
	for _, et := range entityTypes {
		stmts[et] = make(map[entityDBOp]*sql.Stmt)
		for _, dbOp := range entityDBOps {
			stmts[et][dbOp], err = db.Prepare(getQuery(et, dbOp))
			if err != nil {
				return nil, err
			}
		}
	}

	return &postgresStorer{
		idGen:   idGen,
		db:      db,
		dbProxy: dbProxy,
		stmts:   stmts,
	}, nil
}

func (s *postgresStorer) PutEntity(e *api.Entity) error {
	if err := validateEntity(e); err != nil {
		return err
	}
	op := UPDATE
	if added, err := s.maybeAddEntityID(e); err != nil {
		return err
	} else if added {
		op = INSERT
	}
	tx, err := s.dbProxy.Begin()
	if err != nil {
		return err
	}
	et := getEntityType(e)
	if _, err := tx.Stmt(s.stmts[et][op]).Exec(toStmtArgs(e)); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *postgresStorer) GetEntity(entityID string) (*api.Entity, error) {
	// TODO (drausin) validate entityID
	et := getEntityTypeFromID(entityID)
	row := s.stmts[et][SELECT].QueryRow(entityID)
	return fromRow(row, entityID, et)
}

func (s *postgresStorer) Close() error {
	for et := range s.stmts {
		for op := range s.stmts[et] {
			if err := s.stmts[et][op].Close(); err != nil {
				return err
			}
		}
	}
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

var entityDBOps = []entityDBOp{INSERT, UPDATE, SELECT}

func getQuery(et entityType, op entityDBOp) string {
	switch op {
	case INSERT:
		return getInsertQuery(entityTypeTbls[et], entityTypeAttrCols[et])
	case UPDATE:
		return getUpdateQuery(entityTypeTbls[et], entityTypeAttrCols[et])
	case SELECT:
		return getSelectQuery(entityTypeTbls[et], entityTypeAttrCols[et])
	default:
		panic(errUnknownDBOperation)
	}
}

func getInsertQuery(table string, attrCols []string) string {
	valueArgs := make([]interface{}, 1+len(attrCols))
	for i := 0; i < len(attrCols); i++ {
		valueArgs[i] = fmt.Sprintf("$%d", i+1)
	}
	fqTable := fmt.Sprintf("%s.%s", entitySchema, table)
	cols := append([]string{entityIDCol}, attrCols...)
	query, _, err := psql.Insert(fqTable).Columns(cols...).Values(valueArgs...).ToSql()
	errors2.MaybePanic(err) // should never happen
	log.Println(query)
	return query
}

func getUpdateQuery(table string, attrCols []string) string {
	valuePairs := make(map[string]interface{})
	for i, attrCol := range attrCols {
		valuePairs[attrCol] = fmt.Sprintf("$%d", i+2)
	}
	fqTable := fmt.Sprintf("%s.%s", entitySchema, table)
	conditions := sq.Eq{entityIDCol: "$1"}
	query, _, err := psql.Update(fqTable).SetMap(valuePairs).Where(conditions).ToSql()
	errors2.MaybePanic(err) // should never happen
	log.Println(query)
	return query
}

func getSelectQuery(table string, attrCols []string) string {
	fqTable := fmt.Sprintf("%s.%s", entitySchema, table)
	conditions := sq.Eq{entityIDCol: "$1"}
	query, _, err := psql.Select(attrCols...).From(fqTable).Where(conditions).ToSql()
	errors2.MaybePanic(err) // should never happen
	log.Println(query)
	return query
}

type entityType uint

const (
	patient entityType = iota
	office
)

var (
	entitySchema   = "entity"
	entityIDCol    = "entity_id"
	entityTypes    = []entityType{patient, office}
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

func toStmtArgs(e *api.Entity) []interface{} {
	entityIDArg := []interface{}{e.EntityId}
	switch ta := e.TypeAttributes.(type) {
	case *api.Entity_Patient:
		return append(entityIDArg, toPatientStmtArgs(ta.Patient))
	case *api.Entity_Office:
		return append(entityIDArg, toOfficeStmtArgs(ta.Office))
	default:
		panic(errUnknownEntityType)
	}
}

func fromRow(row *sql.Row, entityID string, et entityType) (*api.Entity, error) {
	switch et {
	case patient:
		return fromPatientRow(row, entityID)
	case office:
		return fromOfficeRow(row, entityID)
	default:
		panic(errUnknownEntityType)
	}
}

func fromPatientRow(row *sql.Row, entityID string) (*api.Entity, error) {
	p := &api.Patient{}
	var birthdateISO string
	dest := []interface{}{
		&p.LastName,
		&p.FirstName,
		&p.MiddleName,
		&p.Suffix,
		&birthdateISO,
	}
	if err := row.Scan(dest); err != nil {
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

func toPatientStmtArgs(p *api.Patient) []interface{} {
	return []interface{}{
		p.LastName,
		p.FirstName,
		p.MiddleName,
		p.Birthdate.ISO8601(),
	}
}

func fromOfficeRow(row *sql.Row, entityID string) (*api.Entity, error) {
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

func toOfficeStmtArgs(f *api.Office) []interface{} {
	return []interface{}{
		f.Name,
	}
}
