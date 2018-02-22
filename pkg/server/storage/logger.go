package storage

import "go.uber.org/zap/zapcore"

const (
	logStorageType     = "storage_type"
	logPutQueryTimeout = "put_query_timeout"
	logGetQueryTimeout = "get_query_timeout"
)

// MarshalLogObject writes the parameters to the given object encoder.
func (p *Parameters) MarshalLogObject(oe zapcore.ObjectEncoder) error {
	oe.AddString(logStorageType, p.Type.String())
	oe.AddDuration(logPutQueryTimeout, p.PutQueryTimeout)
	oe.AddDuration(logGetQueryTimeout, p.GetQueryTimeout)
	return nil
}
