package storage

import (
	"strings"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
)

type EntityType int

const NEntityTypes = 2
const (
	Patient EntityType = iota
	Office
)

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

func GetEntityTypeFromID(entityID string) EntityType {
	for i := 0; i < NEntityTypes; i++ {
		et := EntityType(i)
		if strings.HasPrefix(entityID, et.IDPrefix()) {
			return et
		}
	}
	panic(ErrUnknownEntityType)
}
