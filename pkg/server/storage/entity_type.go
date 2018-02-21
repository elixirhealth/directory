package storage

import (
	"strings"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
)

/*
 Adding a new entity type:
 1) In directory.proto, add new message type and add to type_attributes in Entity message type.
 2) Increment nEntityTypes and add to the entityType enum values.
 3) Create a migration adding a table for the new type.
 4) Run the tests, and they will tell you what you need to fix (usually by adding a case for the
 new type in functions below and in postgres.go).
*/

type entityType int

const nEntityTypes = 2
const (
	patient entityType = iota
	office
)

func (et entityType) string() string {
	switch et {
	case patient:
		return "patient"
	case office:
		return "office"
	default:
		panic(errUnknownEntityType)
	}
}

func (et entityType) idPrefix() string {
	switch et {
	case patient:
		return "P"
	case office:
		return "F"
	default:
		panic(errUnknownEntityType)
	}
}

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
	for i := 0; i < nEntityTypes; i++ {
		et := entityType(i)
		if strings.HasPrefix(entityID, et.idPrefix()) {
			return et
		}
	}
	panic(errUnknownEntityType)
}
