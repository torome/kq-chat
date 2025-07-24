package project

import (
	"cjgt/app/bi/cmd/rpc/pb"
	"context"
	"encoding/json"

	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProjectConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateProjectConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProjectConfigLogic {
	return &UpdateProjectConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateProjectConfigLogic) UpdateProjectConfig(req *types.UpdateProjectConfigReq) (*types.UpdateProjectConfigResp, error) {
	jsonBytes, err := json.Marshal(req.Config)
	if err != nil {
		panic(err)
	}

	// 将字节切片转换为字符串
	jsonString := string(jsonBytes)

	_, err = l.svcCtx.BiRpc.UpdateProject(l.ctx, &pb.UpdateProjectReq{
		Id:     req.Id,
		Config: jsonString,
	})
	if err != nil {
		return nil, err
	}

	return &types.UpdateProjectConfigResp{Id: 0}, nil
}
