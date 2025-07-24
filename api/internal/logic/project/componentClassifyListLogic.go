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

type ComponentClassifyListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComponentClassifyListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComponentClassifyListLogic {
	return &ComponentClassifyListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ComponentClassifyListLogic) ComponentClassifyList(req *types.ComponentClassifyListReq) (*types.ComponentClassifyListResp, error) {
	resp, err := l.svcCtx.BiRpc.ComponentClassifyList(l.ctx, &bi.ComponentClassifyListReq{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("Failed to get list"), "Failed to get list err : %v ,req:%+v", err, req)
	}

	var typesUserLogoOrderList []types.ComponentClassifyListView

	if len(resp.List) > 0 {

		for _, logoOrder := range resp.List {

			var typeLogoOrder types.ComponentClassifyListView
			_ = copier.Copy(&typeLogoOrder, logoOrder)
			// id转名称，要进行处理，枚举？ todo...
			// typeLogoOrder.Grade = ""
			typesUserLogoOrderList = append(typesUserLogoOrderList, typeLogoOrder)
		}
	}

	return &types.ComponentClassifyListResp{
		List:  typesUserLogoOrderList,
		Total: resp.Total,
	}, nil
}
