package bot

import (
	"context"

	"ai-agent/internal/svc"
	"ai-agent/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SpeechToTextLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSpeechToTextLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SpeechToTextLogic {
	return &SpeechToTextLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SpeechToTextLogic) SpeechToText(req *types.SpeechToTextReq) (resp *types.SpeechToTextResp, err error) {
	l.Infof("SpeechToText: botId=%s, url=%s, format=%s", req.BotId, req.Url, req.VoiceFormat)

	// 模拟语音识别结果
	resp = &types.SpeechToTextResp{
		Result: "你好，请介绍一下go-zero框架", // 模拟识别结果
	}

	return resp, nil
}
