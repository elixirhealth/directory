package directoryapi

import (
	"strings"
	"time"
)

// TODO add ValidateENDPOINTRequest method for each service ENDPOINT

func (m *Date) ISO8601() string {
	return strings.Join([]string{string(m.Year), string(m.Month), string(m.Day)}, "-")
}

func FromISO8601(isoDate string) (*Date, error) {
	asTime, err := time.Parse("2006-01-02", isoDate)
	if err != nil {
		return nil, err
	}
	return &Date{
		Year:  uint32(asTime.Year()),
		Month: uint32(asTime.Month()),
		Day:   uint32(asTime.Day()),
	}, nil
}
