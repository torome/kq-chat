package bot

import (
	"net/http"

	"ai-agent/common/result"

	"ai-agent/internal/logic/bot"
	"ai-agent/internal/svc"
	"ai-agent/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetChatRecordsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetChatRecordsReq
		if err := httpx.Parse(r, &req); err != nil {
			result.ParamErrorResult(r, w, err)
			return
		}

		l := bot.NewGetChatRecordsLogic(r.Context(), svcCtx)
		resp, err := l.GetChatRecords(&req)
		result.HttpResult(r, w, resp, err)
	}
}
