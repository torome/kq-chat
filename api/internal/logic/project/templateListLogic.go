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

type TemplateListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTemplateListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TemplateListLogic {
	return &TemplateListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TemplateListLogic) TemplateList(req *types.TemplateListReq) (*types.TemplateListResp, error) {
	resp, err := l.svcCtx.BiRpc.TemplateList(l.ctx, &bi.TemplateListReq{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("Failed to get list"), "Failed to get list err : %v ,req:%+v", err, req)
	}

	var typesUserLogoOrderList []types.TemplateListView

	if len(resp.List) > 0 {

		for _, logoOrder := range resp.List {

			var typeLogoOrder types.TemplateListView
			_ = copier.Copy(&typeLogoOrder, logoOrder)
			// id转名称，要进行处理，枚举？ todo...
			// typeLogoOrder.Grade = ""
			typesUserLogoOrderList = append(typesUserLogoOrderList, typeLogoOrder)
		}
	}

	return &types.TemplateListResp{
		List:  typesUserLogoOrderList,
		Total: resp.Total,
	}, nil
}
