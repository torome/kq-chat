package project

import (
	"cjgt/app/bi/cmd/rpc/pb"
	"cjgt/common/xerr"
	"context"
	"github.com/pkg/errors"

	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteProjectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteProjectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteProjectLogic {
	return &DeleteProjectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteProjectLogic) DeleteProject(req *types.DeleteProjectReq) (*types.DeleteProjectResp, error) {
	id := req.Id // 先不考虑批量删除
	_, err := l.svcCtx.BiRpc.DeleteProject(l.ctx, &pb.DeleteProjectReq{
		Id: id,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("数据不存在"), "CreateHomestayOrder homestay no exists id : %d", id)
	}

	return &types.DeleteProjectResp{}, nil
}
