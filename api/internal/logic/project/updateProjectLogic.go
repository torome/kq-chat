package project

import (
	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"
	"cjgt/app/bi/cmd/rpc/pb"
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProjectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateProjectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProjectLogic {
	return &UpdateProjectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateProjectLogic) UpdateProject(req *types.UpdateProjectReq) (*types.UpdateProjectResp, error) {

	_, err := l.svcCtx.BiRpc.UpdateProject(l.ctx, &pb.UpdateProjectReq{
		Id:   req.Id,
		Name: req.Name,
		//TemplateId: 0, // 不允许更改模版
		//Config:   jsonString,
		Remark:   req.Remark,
		RowState: req.RowState,
	})
	if err != nil {
		return nil, err
	}

	return &types.UpdateProjectResp{Id: 0}, nil
}
