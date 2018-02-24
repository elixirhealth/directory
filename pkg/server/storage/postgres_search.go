package storage

import (
	"database/sql"
	"fmt"
	"strings"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
)

var searchers = []searcher{
	&btreeSearcher{
		searcherName: "PatientEntityID",
		et:           patient,
		indexedValue: entityIDCol,
	},
	&trigramSearcher{
		searcherName: "PatientName",
		et:           patient,
		indexedValue: strings.Join([]string{
			nonEmptyUpper(lastNameCol),
			nonEmptyUpper(firstNameCol),
		}, " || ' ' || "),
	},

	&btreeSearcher{
		searcherName: "OfficeEntityID",
		et:           office,
		indexedValue: entityIDCol,
	},
	&trigramSearcher{
		searcherName: "OfficeName",
		et:           office,
		indexedValue: nonEmptyUpper(nameCol),
	},
}

func nonEmptyUpper(colName string) string {
	return fmt.Sprintf("COALESCE(UPPER(%s), '')", colName)
}

type searcher interface {
	entityType() entityType
	name() string
	predicate() string
	similarity() string
	preprocQuery(raw string) string
}

type btreeSearcher struct {
	et            entityType
	searcherName  string
	indexedValue  string
	caseSensitive bool
}

func (ps *btreeSearcher) entityType() entityType {
	return ps.et
}

func (ps *btreeSearcher) name() string {
	return ps.searcherName
}

func (ps *btreeSearcher) predicate() string {
	return ps.indexedValue + " LIKE ?"
}

func (ps *btreeSearcher) similarity() string {
	// since we assume that match occurred, the similarity is the fraction of indexed
	// indexedValue that the prefix matches
	return fmt.Sprintf("char_length(?)::real / char_length(%s)::real AS %s", entityIDCol,
		similarityCol)
}

func (ps *btreeSearcher) preprocQuery(raw string) string {
	if !ps.caseSensitive {
		return strings.ToUpper(raw) + "%"
	}
	return raw + "%"
}

type trigramSearcher struct {
	et            entityType
	searcherName  string
	indexedValue  string
	caseSensitive bool
}

func (ts *trigramSearcher) entityType() entityType {
	return ts.et
}

func (ts *trigramSearcher) name() string {
	return ts.searcherName
}
func (ts *trigramSearcher) predicate() string {
	return ts.indexedValue + " % ?"
}

func (ts *trigramSearcher) similarity() string {
	return fmt.Sprintf("similarity(%s, ?) AS %s", ts.indexedValue, similarityCol)
}

func (ts *trigramSearcher) preprocQuery(raw string) string {
	if !ts.caseSensitive {
		return strings.ToUpper(raw)
	}
	return raw
}

type searchResultMerger interface {
	merge(rows *sql.Rows, searcherName string, et entityType) error
	top(n uint) ([]*api.Entity, error)
}
