package directoryapi

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// TODO add ValidateENDPOINTRequest method for each service ENDPOINT

const (
	isoDateFormat = "2006-01-02"
)

var (
	ErrPatientMissingLastName  = errors.New("patient missing last name")
	ErrPatientMissingFirstName = errors.New("patient missing first name")
	ErrPatientMissingBirthdate = errors.New("patient missing birthdate")

	ErrOfficeMissingName = errors.New("office missing name")

	errUnknownEntityType = errors.New("unknown entity type")
)

func ValidateEntity(e *Entity) error {
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
	if p.LastName == "" {
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

// FromISO8601 parses a *Date from the given string, assumed to be in ISO date or timestamp
// (c.f., time.RFC3339).
func FromISO8601(isoDate string) (*Date, error) {
	asTime, err := time.Parse(isoDateFormat, isoDate)
	if err != nil {
		asTime, err = time.Parse(time.RFC3339, isoDate)
		if err != nil {
			return nil, err
		}
	}
	return &Date{
		Year:  uint32(asTime.Year()),
		Month: uint32(asTime.Month()),
		Day:   uint32(asTime.Day()),
	}, nil
}
