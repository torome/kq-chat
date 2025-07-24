package project

import (
	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"
	"cjgt/app/bi/cmd/rpc/bi"
	"cjgt/common/xerr"
	"context"
	"encoding/json"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProjectDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProjectDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProjectDetailLogic {
	return &ProjectDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProjectDetailLogic) ProjectDetail(req *types.ProjectDetailReq) (*types.ProjectDetailResp, error) {
	resp, err := l.svcCtx.BiRpc.ProjectDetail(l.ctx, &bi.ProjectDetailReq{
		Id: req.Id,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
	}

	var typesOrderDetail types.Project
	if resp.Project != nil {
		var jsonData json.RawMessage
		err = copier.Copy(&typesOrderDetail, resp.Project)
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
		}

		if err = json.Unmarshal([]byte(resp.Project.Config), &jsonData); err != nil {
			return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
		}
		typesOrderDetail.Config = jsonData

		return &types.ProjectDetailResp{
			Row: typesOrderDetail,
		}, nil
	}

	return nil, nil
}
