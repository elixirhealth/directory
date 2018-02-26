package postgres

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/drausin/libri/libri/common/errors"
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/directory/pkg/server/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logEntityType = "entity_type"
	logQuery      = "query"
	logSQL        = "sql"
	logArgs       = "args"
	logEntityID   = "entity_id"
	logInsert     = "insert"
	logUpdate     = "update"
	logSearcher   = "searcher"
	logLimit      = "limit"
	logResults    = "results"
)

func logGetSelect(q sq.SelectBuilder, et storage.EntityType, entityID string) []zapcore.Field {
	qSQL, args, err := q.ToSql()
	errors.MaybePanic(err)
	return []zapcore.Field{
		zap.Stringer(logEntityType, et),
		zap.String(logEntityID, entityID),
		zap.String(logSQL, qSQL),
		zap.Array(logArgs, queryArgs(args)),
	}
}

func logSearchSelect(q sq.SelectBuilder, s searcher, query string) []zapcore.Field {
	qSQL, args, err := q.ToSql()
	errors.MaybePanic(err)
	return []zapcore.Field{
		zap.String(logQuery, query),
		zap.String(logSearcher, s.name()),
		zap.Stringer(logEntityType, s.entityType()),
		zap.String(logSQL, qSQL),
		zap.Array(logArgs, queryArgs(args)),
	}
}

func logSearcherFinished(s searcher, query string) []zapcore.Field {
	return []zapcore.Field{
		zap.String(logQuery, query),
		zap.String(logSearcher, s.name()),
		zap.Stringer(logEntityType, s.entityType()),
	}
}

func logSearchRanked(query string, limit uint, ess storage.EntitySims) []zapcore.Field {
	return []zapcore.Field{
		zap.String(logQuery, query),
		zap.Uint(logLimit, limit),
		zap.Array(logResults, ess),
	}
}

func logPutInsert(q sq.InsertBuilder, e *api.Entity) []zapcore.Field {
	qSQL, args, err := q.ToSql()
	errors.MaybePanic(err)
	return []zapcore.Field{
		zap.Stringer(logEntityType, storage.GetEntityType(e)),
		zap.String(logSQL, qSQL),
		zap.Array(logArgs, queryArgs(args)),
	}
}

func logPutResult(entityID string, insert bool) []zapcore.Field {
	return []zapcore.Field{
		zap.String(logEntityID, entityID),
		zap.Bool(logInsert, insert),
		zap.Bool(logUpdate, !insert),
	}
}

func logPutUpdate(q sq.UpdateBuilder, e *api.Entity) []zapcore.Field {
	qSQL, args, err := q.ToSql()
	errors.MaybePanic(err)
	return []zapcore.Field{
		zap.Stringer(logEntityType, storage.GetEntityType(e)),
		zap.String(logSQL, qSQL),
		zap.Array(logArgs, queryArgs(args)),
	}
}

type queryArgs []interface{}

func (qas queryArgs) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, qa := range qas {
		switch val := qa.(type) {
		case string:
			enc.AppendString(val)
		default:
			if err := enc.AppendReflected(qa); err != nil {
				return err
			}
		}
	}
	return nil
}
