package websocket

import (
	"net/http"

	"ai-agent/common/result"

	"ai-agent/internal/logic/websocket"
	"ai-agent/internal/svc"
)

func WebsocketHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := websocket.NewWebsocketLogic(r.Context(), svcCtx)
		err := l.Websocket(w, r)
		if err != nil {
			result.HttpResult(r, w, nil, err)
		}
	}
}
