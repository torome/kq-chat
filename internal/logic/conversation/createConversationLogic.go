package conversation

import (
	"context"
	"fmt"
	"time"

	"ai-agent/internal/svc"
	"ai-agent/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateConversationLogic {
	return &CreateConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateConversationLogic) CreateConversation(req *types.CreateConversationReq) (resp *types.CreateConversationResp, err error) {
	l.Infof("CreateConversation: botId=%s", req.BotId)

	// 生成新的会话ID和标题
	conversationId := fmt.Sprintf("conv_%d", time.Now().UnixNano())
	title := fmt.Sprintf("对话_%s", time.Now().Format("01-02 15:04"))

	resp = &types.CreateConversationResp{
		ConversationId: conversationId,
		Title:          title,
	}

	return resp, nil
}
