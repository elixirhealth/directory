package directoryapi

import (
	"fmt"
	"time"
)

// TODO add ValidateENDPOINTRequest method for each service ENDPOINT

func (m *Date) ISO8601() string {
	return fmt.Sprintf("%d-%d-%d", m.Year, m.Month, m.Day)
}

func FromISO8601(isoDate string) (*Date, error) {
	asTime, err := time.Parse(time.RFC3339, isoDate)
	if err != nil {
		return nil, err
	}
	return &Date{
		Year:  uint32(asTime.Year()),
		Month: uint32(asTime.Month()),
		Day:   uint32(asTime.Day()),
	}, nil
}
