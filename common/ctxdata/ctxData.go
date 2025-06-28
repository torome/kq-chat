package ctxdata

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"strconv"
)

// CtxKeyJwtUserId get uid from ctx
var CtxKeyJwtUserId = "jwtUserId"
var CtxKeyTenantId = "tenantId"

func GetUidFromCtx(ctx context.Context) int64 {
	var uid int64
	if jsonUid, ok := ctx.Value(CtxKeyJwtUserId).(json.Number); ok {
		if int64Uid, err := jsonUid.Int64(); err == nil {
			uid = int64Uid
		} else {
			logx.WithContext(ctx).Errorf("GetUidFromCtx err : %+v", err)
		}
	}
	return uid
}

func GetTenantIdFromCtx(ctx context.Context) int64 {
	var uid int64
	d := ctx.Value(CtxKeyTenantId)
	fmt.Printf("CtxKeyTenantId: %v", d)
	if jsonUid, ok := ctx.Value(CtxKeyTenantId).(string); ok {
		if int64Uid, err := strconv.ParseInt(jsonUid, 10, 64); err == nil {
			fmt.Printf("------ *** ------- %d", int64Uid)
			uid = int64Uid
		} else {
			logx.WithContext(ctx).Errorf("GetTenantIdFromCtx err : %+v", err)
		}
	}
	return uid
}
