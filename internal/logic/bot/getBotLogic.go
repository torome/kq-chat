package bot

import (
	"context"

	"ai-agent/internal/svc"
	"ai-agent/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBotLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBotLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBotLogic {
	return &GetBotLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetBotLogic) GetBot(req *types.GetBotReq) (resp *types.GetBotResp, err error) {
	// 模拟Bot数据
	bot := &types.GetBotResp{
		Code:    0,
		Message: "success",
		BotInfo: types.BotInfo{
			BotId:          req.BotId,
			Name:           "智能助手",
			Avatar:         "https://example.com/avatar.png",
			WelcomeMessage: "你好，有什么我可以帮到你？",
			InitQuestions: []string{
				"介绍一下go-zero框架",
				"什么是微服务架构",
				"如何实现WebSocket通信",
			},
			IsNeedRecommend:         true,
			MultiConversationEnable: true,
			SearchEnable:            true,
			SearchFileEnable:        true,
			VoiceSettings: &types.VoiceSettings{
				Enable:     true,
				InputType:  "iat_standard",
				OutputType: "xiaoyun",
			},
		},
	}

	return bot, nil
}
