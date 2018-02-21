package directoryapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO add TestValidateENDPOINTRequest method for each service ENDPOINT

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

func TestFromISO8601(t *testing.T) {
	cases := []struct {
		from     string
		expected *Date
	}{
		{from: "2006-01-02", expected: &Date{Year: 2006, Month: 1, Day: 2}},
		{from: "2006-01-02T15:04:05Z", expected: &Date{Year: 2006, Month: 1, Day: 2}},
	}
	for _, c := range cases {
		d, err := FromISO8601(c.from)
		assert.Nil(t, err, c.from)
		assert.Equal(t, c.expected, d, c.from)
	}

	d, err := FromISO8601("1/2/2006")
	assert.NotNil(t, err)
	assert.Nil(t, d)
}
