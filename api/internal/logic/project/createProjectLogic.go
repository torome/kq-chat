package project

import (
	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"
	"cjgt/app/bi/cmd/rpc/pb"
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateProjectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateProjectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateProjectLogic {
	return &CreateProjectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateProjectLogic) CreateProject(req *types.CreateProjectReq) (*types.CreateProjectResp, error) {

	// 创建项目，不涉及配置信息
	resp, err := l.svcCtx.BiRpc.CreateProject(l.ctx, &pb.CreateProjectReq{
		Name:       req.Name,
		TemplateId: req.TemplateId,
		Config:     "{}",
		Remark:     req.Remark,
		RowState:   0, // 创建项目，默认未上架，需要手动进行上架
	})
	if err != nil {
		return nil, err
	}

	return &types.CreateProjectResp{Id: resp.Id}, nil
}
