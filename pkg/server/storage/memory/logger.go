package memory

import (
	"github.com/elxirhealth/directory/pkg/server/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logQuery    = "query"
	logEntityID = "entity_id"
	logInsert   = "insert"
	logUpdate   = "update"
	logLimit    = "limit"
	logResults  = "results"
)

func logPutResult(entityID string, insert bool) []zapcore.Field {
	return []zapcore.Field{
		zap.String(logEntityID, entityID),
		zap.Bool(logInsert, insert),
		zap.Bool(logUpdate, !insert),
	}
}

func logSearchRanked(query string, limit uint, ess storage.EntitySims) []zapcore.Field {
	return []zapcore.Field{
		zap.String(logQuery, query),
		zap.Uint(logLimit, limit),
		zap.Array(logResults, ess),
	}
}
