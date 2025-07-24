package datav

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

type DatasetSelectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDatasetSelectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DatasetSelectLogic {
	return &DatasetSelectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DatasetSelectLogic) DatasetSelect(req *types.DatasetSelectReq) (*types.DatasetSelectResp, error) {
	resp, err := l.svcCtx.BiRpc.DatasetList(l.ctx, &bi.DatasetListReq{
		ProjectId: req.ProjectId,
		Page:      1,
		PageSize:  100,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("Failed to get list"), "Failed to get list err : %v ,req:%+v", err, req)
	}

	var typesUserLogoOrderList []types.DatasetListView

	if len(resp.List) > 0 {

		for _, logoOrder := range resp.List {

			var typeLogoOrder types.DatasetListView
			_ = copier.Copy(&typeLogoOrder, logoOrder)

			typesUserLogoOrderList = append(typesUserLogoOrderList, typeLogoOrder)
		}
	}

	return &types.DatasetSelectResp{
		List: typesUserLogoOrderList,
	}, nil
}
