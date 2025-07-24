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

type TemplateDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTemplateDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TemplateDetailLogic {
	return &TemplateDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TemplateDetailLogic) TemplateDetail(req *types.TemplateDetailReq) (*types.TemplateDetailResp, error) {
	resp, err := l.svcCtx.BiRpc.TemplateDetail(l.ctx, &bi.TemplateDetailReq{
		Id: req.Id,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
	}

	var typesOrderDetail types.Template
	if resp.Template != nil {
		var jsonData json.RawMessage
		err = copier.Copy(&typesOrderDetail, resp.Template)
		if err = json.Unmarshal([]byte(resp.Template.Config), &jsonData); err != nil {
			return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
		}
		typesOrderDetail.Config = jsonData
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
		}
		return &types.TemplateDetailResp{
			Row: typesOrderDetail,
		}, nil
	}

	return nil, nil
}
