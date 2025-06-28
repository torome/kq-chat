package bot

import (
	"net/http"

	"ai-agent/common/result"

	"ai-agent/internal/logic/bot"
	"ai-agent/internal/svc"
	"ai-agent/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func SpeechToTextHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SpeechToTextReq
		if err := httpx.Parse(r, &req); err != nil {
			result.ParamErrorResult(r, w, err)
			return
		}

		l := bot.NewSpeechToTextLogic(r.Context(), svcCtx)
		resp, err := l.SpeechToText(&req)
		result.HttpResult(r, w, resp, err)
	}
}
