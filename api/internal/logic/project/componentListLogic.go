package project

import (
	"cjgt/app/bi/cmd/rpc/bi"
	"cjgt/common/xerr"
	"context"
	"encoding/json"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComponentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComponentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComponentListLogic {
	return &ComponentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ComponentListLogic) ComponentList(req *types.ComponentListReq) (*types.ComponentListResp, error) {
	resp, err := l.svcCtx.BiRpc.ComponentList(l.ctx, &bi.ComponentListReq{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("Failed to get list"), "Failed to get list err : %v ,req:%+v", err, req)
	}

	var typesUserLogoOrderList []types.ComponentListView

	if len(resp.List) > 0 {

		for _, logoOrder := range resp.List {

			var typeLogoOrder types.ComponentListView
			_ = copier.Copy(&typeLogoOrder, logoOrder)
			// id转名称，要进行处理，枚举？ todo...
			// typeLogoOrder.Grade = ""
			var jsonData json.RawMessage
			if err = json.Unmarshal([]byte(logoOrder.Config), &jsonData); err != nil {
				//log.Fatal(err)
				return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v ", err)
			}
			typeLogoOrder.Config = jsonData

			typesUserLogoOrderList = append(typesUserLogoOrderList, typeLogoOrder)
		}
	}

	return &types.ComponentListResp{
		List:  typesUserLogoOrderList,
		Total: resp.Total,
	}, nil
}
