package directoryapi

import (
	"fmt"

	"github.com/pkg/errors"
)

// TODO add ValidateENDPOINTRequest method for each service ENDPOINT

var (
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

// ISO8601 returns the YYYY-MM-DD ISO 8601 date string.
func (m *Date) ISO8601() string {
	return fmt.Sprintf("%04d-%02d-%02d", m.Year, m.Month, m.Day)
}
