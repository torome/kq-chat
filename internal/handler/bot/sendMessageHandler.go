package bot

import (
	"net/http"

	"ai-agent/common/result"

	"ai-agent/internal/logic/bot"
	"ai-agent/internal/svc"
	"ai-agent/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func SendMessageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendMessageReq
		if err := httpx.Parse(r, &req); err != nil {
			result.ParamErrorResult(r, w, err)
			return
		}

		l := bot.NewSendMessageLogic(r.Context(), svcCtx)
		err := l.SendMessage(&req)
		result.HttpResult(r, w, nil, err)
	}
}
