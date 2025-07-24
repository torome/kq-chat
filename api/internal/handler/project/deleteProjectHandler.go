package project

import (
	"net/http"

	"cjgt/common/result"

	"cjgt/app/bi/cmd/api/internal/logic/project"
	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeleteProjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeleteProjectReq
		if err := httpx.Parse(r, &req); err != nil {
			result.ParamErrorResult(r, w, err)
			return
		}

		l := project.NewDeleteProjectLogic(r.Context(), svcCtx)
		resp, err := l.DeleteProject(&req)
		result.HttpResult(r, w, resp, err)
	}
}
