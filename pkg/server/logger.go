package server

import (
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logEntityID   = "entity_id"
	logNewEntity  = "new_entity"
	logEntityType = "entity_type"
	logStorage    = "storage"
	logDBUrl      = "db_url"
)

func logPutEntityRq(rq *api.PutEntityRequest) []zapcore.Field {
	if rq.Entity == nil {
		return []zapcore.Field{}
	}
	return []zapcore.Field{
		zap.String(logEntityID, rq.Entity.EntityId),
		zap.Bool(logNewEntity, rq.Entity.EntityId == ""),
		zap.String(logEntityType, rq.Entity.Type()),
	}
}

func logPutEntityRp(rq *api.PutEntityRequest, rp *api.PutEntityResponse) []zapcore.Field {
	return []zapcore.Field{
		zap.String(logEntityID, rp.EntityId),
		zap.Bool(logNewEntity, rq.Entity.EntityId == ""),
		zap.String(logEntityType, rq.Entity.Type()),
	}
}

func logGetEntityRp(rp *api.GetEntityResponse) []zapcore.Field {
	return []zapcore.Field{
		zap.String(logEntityID, rp.Entity.EntityId),
		zap.String(logEntityType, rp.Entity.Type()),
	}
}
