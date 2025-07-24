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

type ProjectListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProjectListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProjectListLogic {
	return &ProjectListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProjectListLogic) ProjectList(req *types.ProjectListReq) (*types.ProjectListResp, error) {
	resp, err := l.svcCtx.BiRpc.ProjectList(l.ctx, &bi.ProjectListReq{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("Failed to get user logo order list"), "Failed to get user logo order list err : %v ,req:%+v", err, req)
	}

	var typesUserLogoOrderList []types.ProjectListView

	if len(resp.List) > 0 {

		for _, logoOrder := range resp.List {

			var typeLogoOrder types.ProjectListView
			_ = copier.Copy(&typeLogoOrder, logoOrder)
			// id转名称，要进行处理，枚举？ todo...
			// typeLogoOrder.Grade = ""
			typesUserLogoOrderList = append(typesUserLogoOrderList, typeLogoOrder)
		}
	}

	return &types.ProjectListResp{
		List:  typesUserLogoOrderList,
		Total: resp.Total,
	}, nil
}
