package directoryapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var okEntity = &Entity{
	TypeAttributes: &Entity_Patient{
		Patient: &Patient{
			LastName:  "Last Name",
			FirstName: "First Name",
			Birthdate: &Date{Year: 2006, Month: 1, Day: 2},
		},
	},
}

func TestValidatePutEntityRequest(t *testing.T) {
	cases := map[string]struct {
		rq       *PutEntityRequest
		expected error
	}{
		"ok": {
			rq: &PutEntityRequest{
				Entity: okEntity,
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
			e:        okEntity,
			expected: nil,
		},
		"missing type attributes": {
			e:        &Entity{},
			expected: ErrMissingTypeAttributes,
		},
		"patient missing last name": {
			e: &Entity{
				TypeAttributes: &Entity_Patient{
					Patient: &Patient{
						FirstName: "First Name",
						Birthdate: &Date{Year: 2006, Month: 1, Day: 2},
					},
				},
			},
			expected: ErrPatientMissingLastName,
		},
		"patient missing first name": {
			e: &Entity{
				TypeAttributes: &Entity_Patient{
					Patient: &Patient{
						LastName:  "Last Name",
						Birthdate: &Date{Year: 2006, Month: 1, Day: 2},
					},
				},
			},
			expected: ErrPatientMissingFirstName,
		},
		"patient missing birthdate": {
			e: &Entity{
				TypeAttributes: &Entity_Patient{
					Patient: &Patient{
						LastName:  "Last Name",
						FirstName: "First Name",
					},
				},
			},
			expected: ErrPatientMissingBirthdate,
		},
		"office missing name": {
			e: &Entity{
				TypeAttributes: &Entity_Office{
					Office: &Office{},
				},
			},
			expected: ErrOfficeMissingName,
		},
	}

	for desc, c := range cases {
		err := ValidateEntity(c.e)
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
