package directoryapi

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// ErrPutMissingEntity denotes when a Put request is missing the Entity object.
	ErrPutMissingEntity = errors.New("put request missing entity")

	// ErrGetMissingEntityID denotes when a get request is missing the entity ID.
	ErrGetMissingEntityID = errors.New("get request missing entity ID")

	// ErrMissingTypeAttributes denotes when an entity is missing the expected type_attributes
	// field.
	ErrMissingTypeAttributes = errors.New("entity missing type_attributes")

	// ErrPatientMissingLastName denotes when a patient entity is missing the last name.
	ErrPatientMissingLastName = errors.New("patient missing last name")

	// ErrPatientMissingFirstName denotes when a patient entity is missing the first name.
	ErrPatientMissingFirstName = errors.New("patient missing first name")

	// ErrPatientMissingBirthdate denotes when a patient entity is missing the birthdate.
	ErrPatientMissingBirthdate = errors.New("patient missing birthdate")

	// ErrOfficeMissingName denotes when an office entity is missing the name.
	ErrOfficeMissingName = errors.New("office missing name")

	errUnknownEntityType = errors.New("unknown entity type")
)

// ValidatePutEntityRequest checks that the PutEntityRequest has the required fields populated.
func ValidatePutEntityRequest(rq *PutEntityRequest) error {
	if rq.Entity == nil {
		return ErrPutMissingEntity
	}
	return ValidateEntity(rq.Entity)
}

// ValidateGetEntityRequest checks that the GetEntityRequest has the required fields populated.
func ValidateGetEntityRequest(rq *GetEntityRequest) error {
	if rq.EntityId == "" {
		return ErrGetMissingEntityID
	}
	return nil
}

// ValidateEntity validates that the entity has the expected fields populated given its type. It
// does not validate that the EntityId is present or of any particular form.
func ValidateEntity(e *Entity) error {
	if e.TypeAttributes == nil {
		return ErrMissingTypeAttributes
	}
	switch ta := e.TypeAttributes.(type) {
	case *Entity_Patient:
		return validatePatient(ta.Patient)
	case *Entity_Office:
		return validateOffice(ta.Office)
	}
	panic(errUnknownEntityType)
}

func validatePatient(p *Patient) error {
	if p.LastName == "" {
		return ErrPatientMissingLastName
	}
	if p.FirstName == "" {
		return ErrPatientMissingFirstName
	}
	if p.Birthdate == nil {
		return ErrPatientMissingBirthdate
	}
	return nil
}

func validateOffice(p *Office) error {
	if p.Name == "" {
		return ErrOfficeMissingName
	}
	return nil
}

// Type returns a string descriptor of the entity type.
func (m *Entity) Type() string {
	switch m.TypeAttributes.(type) {
	case *Entity_Patient:
		return "patient"
	case *Entity_Office:
		return "office"
	default:
		panic(errUnknownEntityType)
	}
}

// ISO8601 returns the YYYY-MM-DD ISO 8601 date string.
func (m *Date) ISO8601() string {
	return fmt.Sprintf("%04d-%02d-%02d", m.Year, m.Month, m.Day)
}
