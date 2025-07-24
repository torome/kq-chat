package project

import (
	"cjgt/app/bi/cmd/rpc/bi"
	"cjgt/common/xerr"
	"context"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComponentClassifyDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComponentClassifyDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComponentClassifyDetailLogic {
	return &ComponentClassifyDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ComponentClassifyDetailLogic) ComponentClassifyDetail(req *types.ComponentClassifyDetailReq) (*types.ComponentClassifyDetailResp, error) {
	resp, err := l.svcCtx.BiRpc.ComponentClassifyDetail(l.ctx, &bi.ComponentClassifyDetailReq{
		Id: req.Id,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
	}

	var typesOrderDetail types.ComponentClassify
	if resp.ComponentClassify != nil {

		err = copier.Copy(&typesOrderDetail, resp.ComponentClassify)
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %d", err, req.Id)
		}
		return &types.ComponentClassifyDetailResp{
			Row: typesOrderDetail,
		}, nil
	}

	return nil, nil
}
