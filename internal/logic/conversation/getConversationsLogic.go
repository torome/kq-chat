package conversation

import (
	"context"
	"time"

	"ai-agent/internal/svc"
	"ai-agent/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationsLogic {
	return &GetConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConversationsLogic) GetConversations(req *types.GetConversationsReq) (resp *types.GetConversationsResp, err error) {
	l.Infof("GetConversations: botId=%s, limit=%d, offset=%d, isDefault=%t",
		req.BotId, req.Limit, req.Offset, req.IsDefault)

	var conversations []types.ConversationInfo

	if req.IsDefault {
		// 返回默认会话
		conversations = []types.ConversationInfo{
			{
				ConversationId: "default_conv",
				Title:          "默认对话",
				CreateTime:     time.Now().Add(-24 * time.Hour).Format("2006-01-02T15:04:05Z"),
				UpdateTime:     time.Now().Add(-1 * time.Hour).Format("2006-01-02T15:04:05Z"),
				IsDefault:      true,
			},
		}
	} else {
		// 返回普通会话列表
		allConversations := []types.ConversationInfo{
			{
				ConversationId: "conv_001",
				Title:          "go-zero框架讨论",
				CreateTime:     time.Now().Add(-2 * time.Hour).Format("2006-01-02T15:04:05Z"),
				UpdateTime:     time.Now().Add(-30 * time.Minute).Format("2006-01-02T15:04:05Z"),
				IsDefault:      false,
			},
			{
				ConversationId: "conv_002",
				Title:          "微服务架构设计",
				CreateTime:     time.Now().Add(-5 * time.Hour).Format("2006-01-02T15:04:05Z"),
				UpdateTime:     time.Now().Add(-1 * time.Hour).Format("2006-01-02T15:04:05Z"),
				IsDefault:      false,
			},
			{
				ConversationId: "conv_003",
				Title:          "WebSocket实现方案",
				CreateTime:     time.Now().Add(-1 * 24 * time.Hour).Format("2006-01-02T15:04:05Z"),
				UpdateTime:     time.Now().Add(-2 * time.Hour).Format("2006-01-02T15:04:05Z"),
				IsDefault:      false,
			},
			{
				ConversationId: "conv_004",
				Title:          "数据库设计优化",
				CreateTime:     time.Now().Add(-3 * 24 * time.Hour).Format("2006-01-02T15:04:05Z"),
				UpdateTime:     time.Now().Add(-6 * time.Hour).Format("2006-01-02T15:04:05Z"),
				IsDefault:      false,
			},
			{
				ConversationId: "conv_005",
				Title:          "API接口设计规范",
				CreateTime:     time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02T15:04:05Z"),
				UpdateTime:     time.Now().Add(-1 * 24 * time.Hour).Format("2006-01-02T15:04:05Z"),
				IsDefault:      false,
			},
		}

		// 分页处理
		start := req.Offset
		end := req.Offset + req.Limit
		if start >= len(allConversations) {
			start = len(allConversations)
		}
		if end > len(allConversations) {
			end = len(allConversations)
		}

		conversations = allConversations[start:end]
	}

	resp = &types.GetConversationsResp{
		Data:  conversations,
		Total: len(conversations),
	}

	return resp, nil
}
