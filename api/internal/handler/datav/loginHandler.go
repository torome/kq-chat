package datav

import (
	"net/http"

	"cjgt/common/result"

	"cjgt/app/bi/cmd/api/internal/logic/datav"
	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := httpx.Parse(r, &req); err != nil {
			result.ParamErrorResult(r, w, err)
			return
		}

		l := datav.NewLoginLogic(r.Context(), svcCtx)
		resp, err := l.Login(&req)
		result.HttpResult(r, w, resp, err)
	}
}
