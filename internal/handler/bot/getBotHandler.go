package bot

import (
	"net/http"

	"ai-agent/common/result"

	"ai-agent/internal/logic/bot"
	"ai-agent/internal/svc"
	"ai-agent/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetBotHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetBotReq
		if err := httpx.Parse(r, &req); err != nil {
			result.ParamErrorResult(r, w, err)
			return
		}

		l := bot.NewGetBotLogic(r.Context(), svcCtx)
		resp, err := l.GetBot(&req)
		result.HttpResult(r, w, resp, err)
	}
}
