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

type ComponentDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComponentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComponentDetailLogic {
	return &ComponentDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ComponentDetailLogic) ComponentDetail(req *types.ComponentDetailReq) (*types.ComponentDetailResp, error) {
	resp, err := l.svcCtx.BiRpc.ComponentDetail(l.ctx, &bi.ComponentDetailReq{
		Id: req.Id,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
	}

	var typesOrderDetail types.Component
	if resp.Component != nil {
		var jsonData json.RawMessage
		err = copier.Copy(&typesOrderDetail, resp.Component)
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
		}
		if err = json.Unmarshal([]byte(resp.Component.Config), &jsonData); err != nil {
			//log.Fatal(err)
			return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
		}
		typesOrderDetail.Config = jsonData
		return &types.ComponentDetailResp{
			Row: typesOrderDetail,
		}, nil
	}

	return nil, nil
}
