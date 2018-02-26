package storage

import (
	"strings"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
)

// EntityType is an enum for different types of entities.
type EntityType int

// NEntityTypes defines the number of entity types.
const NEntityTypes = 2

const (
	// Patient identities a patient entity type.
	Patient EntityType = iota

	// Office identifies an office entity type.
	Office
)

// String returns a string representation for the entity type.
func (et EntityType) String() string {
	switch et {
	case Patient:
		return "Patient"
	case Office:
		return "Office"
	default:
		panic(ErrUnknownEntityType)
	}
}

// IDPrefix returns the prefix to use in constructing an ID for the entity type.
func (et EntityType) IDPrefix() string {
	switch et {
	case Patient:
		return "P"
	case Office:
		return "F"
	default:
		panic(ErrUnknownEntityType)
	}
}

// GetEntityType returns the EntityType for the given *api.Entity.
func GetEntityType(e *api.Entity) EntityType {
	switch e.TypeAttributes.(type) {
	case *api.Entity_Patient:
		return Patient
	case *api.Entity_Office:
		return Office
	default:
		panic(ErrUnknownEntityType)
	}
}

// GetEntityTypeFromID infers the EntityType from the prefix of the entity ID.
func GetEntityTypeFromID(entityID string) EntityType {
	for i := 0; i < NEntityTypes; i++ {
		et := EntityType(i)
		if strings.HasPrefix(entityID, et.IDPrefix()) {
			return et
		}
	}
	panic(ErrUnknownEntityType)
}
