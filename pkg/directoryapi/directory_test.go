package directoryapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntity_Type(t *testing.T) {
	cases := []struct {
		e        *Entity
		expected string
	}{
		{e: NewTestPatient(0, true), expected: "PATIENT"},
		{e: NewTestOffice(0, true), expected: "OFFICE"},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, c.e.Type())
	}
}

func TestEntity_Name(t *testing.T) {
	cases := []struct {
		e        *Entity
		expected string
	}{
		{e: NewTestPatient(0, true), expected: "First Name 0 Last Name 0"},
		{e: NewTestOffice(0, true), expected: "Office Name 0"},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, c.e.Name())
	}
}

func TestValidatePutEntityRequest(t *testing.T) {
	cases := map[string]struct {
		rq       *PutEntityRequest
		expected error
	}{
		"ok": {
			rq: &PutEntityRequest{
				Entity: NewTestPatient(0, false),
			},
			expected: nil,
		},
		"invalid entity": {
			rq: &PutEntityRequest{
				Entity: &Entity{},
			},
			expected: ErrMissingTypeAttributes,
		},
	}

	for desc, c := range cases {
		err := ValidatePutEntityRequest(c.rq)
		assert.Equal(t, c.expected, err, desc)
	}
}

func TestValidateGetEntityRequest(t *testing.T) {
	cases := map[string]struct {
		rq       *GetEntityRequest
		expected error
	}{
		"ok": {
			rq:       &GetEntityRequest{EntityId: "some entity ID"},
			expected: nil,
		},
		"missing entity ID": {
			rq:       &GetEntityRequest{},
			expected: ErrGetMissingEntityID,
		},
	}

	for desc, c := range cases {
		err := ValidateGetEntityRequest(c.rq)
		assert.Equal(t, c.expected, err, desc)
	}
}

func TestValidateEntity(t *testing.T) {
	cases := map[string]struct {
		e        *Entity
		expected error
	}{
		"ok": {
			e:        NewTestPatient(0, false),
			expected: nil,
		},
		"missing type attributes": {
			e:        &Entity{},
			expected: ErrMissingTypeAttributes,
		},
		"patient missing last name": {
			e: NewPatient("", &Patient{
				FirstName: "First Name",
				Birthdate: &Date{Year: 2006, Month: 1, Day: 2},
			}),
			expected: ErrPatientMissingLastName,
		},
		"patient missing first name": {
			e: NewPatient("", &Patient{
				LastName:  "Last Name",
				Birthdate: &Date{Year: 2006, Month: 1, Day: 2},
			}),
			expected: ErrPatientMissingFirstName,
		},
		"patient missing birthdate": {
			e: NewPatient("", &Patient{
				LastName:  "Last Name",
				FirstName: "First Name",
			}),
			expected: ErrPatientMissingBirthdate,
		},
		"office missing name": {
			e:        NewOffice("", &Office{}),
			expected: ErrOfficeMissingName,
		},
	}

	for desc, c := range cases {
		err := ValidateEntity(c.e)
		assert.Equal(t, c.expected, err, desc)
	}
}

func TestValidateSearchEntityRequest(t *testing.T) {
	cases := map[string]struct {
		rq       *SearchEntityRequest
		expected error
	}{
		"ok": {
			rq: &SearchEntityRequest{
				Query: strings.Repeat("A", 4),
				Limit: 1,
			},
			expected: nil,
		},
		"query too short": {
			rq: &SearchEntityRequest{
				Query: strings.Repeat("A", 3),
				Limit: 1,
			},
			expected: ErrSearchQueryTooShort,
		},
		"query too long": {
			rq: &SearchEntityRequest{
				Query: strings.Repeat("A", 33),
				Limit: 1,
			},
			expected: ErrSearchQueryTooLong,
		},
		"limit too small": {
			rq: &SearchEntityRequest{
				Query: strings.Repeat("A", 4),
				Limit: 0,
			},
			expected: ErrSearchLimitTooSmall,
		},
		"limit too large": {
			rq: &SearchEntityRequest{
				Query: strings.Repeat("A", 4),
				Limit: 9,
			},
			expected: ErrSearchLimitTooLarge,
		},
	}
	for desc, c := range cases {
		err := ValidateSearchEntityRequest(c.rq)
		assert.Equal(t, c.expected, err, desc)
	}
}

func TestDate_ISO8601(t *testing.T) {
	cases := []struct {
		d        *Date
		expected string
	}{
		{d: &Date{Year: 2006, Month: 1, Day: 2}, expected: "2006-01-02"},
		{d: &Date{Year: 2006, Month: 11, Day: 2}, expected: "2006-11-02"},
		{d: &Date{Year: 2006, Month: 11, Day: 12}, expected: "2006-11-12"},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, c.d.ISO8601())
	}
}
