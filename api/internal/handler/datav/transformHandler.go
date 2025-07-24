package datav

import (
	"net/http"

	"cjgt/common/result"

	"cjgt/app/bi/cmd/api/internal/logic/datav"
	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func TransformHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TransformReq
		if err := httpx.Parse(r, &req); err != nil {
			result.ParamErrorResult(r, w, err)
			return
		}

		l := datav.NewTransformLogic(r.Context(), svcCtx)
		resp, err := l.Transform(&req)
		result.HttpResult(r, w, resp, err)
	}
}
