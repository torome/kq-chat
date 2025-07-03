package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ CircleModel = (*customCircleModel)(nil)

type (
	// CircleModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCircleModel.
	CircleModel interface {
		circleModel
	}

	customCircleModel struct {
		*defaultCircleModel
	}
)

// NewCircleModel returns a model for the database table.
func NewCircleModel(conn sqlx.SqlConn, c cache.CacheConf) CircleModel {
	return &customCircleModel{
		defaultCircleModel: newCircleModel(conn, c),
	}
}
