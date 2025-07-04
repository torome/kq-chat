package bot

import (
	"context"
	"time"

	"ai-agent/internal/svc"
	"ai-agent/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetChatRecordsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetChatRecordsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChatRecordsLogic {
	return &GetChatRecordsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetChatRecordsLogic) GetChatRecords(req *types.GetChatRecordsReq) (resp *types.GetChatRecordsResp, err error) {
	//userId := ctxdata.GetUidFromCtx(l.ctx)
	//req.ConversationId = fmt.Sprintf("user_record_%d", userId)
	//l.Infof("GetChatRecords: botId=%s, page=%d, size=%d, sort=%s, conversationId=%s",
	//	req.BotId, req.PageNumber, req.PageSize, req.Sort, req.ConversationId)

	// 模拟聊天记录数据
	records := []types.ChatRecord{
		{
			RecordId:   "record_001",
			Role:       "user",
			Content:    "你好",
			CreateTime: time.Now().Add(-10 * time.Minute).Format("2006-01-02 15:04:05"),
			UpdateTime: time.Now().Add(-10 * time.Minute).Format("2006-01-02 15:04:05"),
		},
		{
			RecordId:   "record_002",
			Role:       "assistant",
			Content:    "您好！我是智能助手，很高兴为您服务。有什么我可以帮助您的吗？",
			CreateTime: time.Now().Add(-9 * time.Minute).Format("2006-01-02 15:04:05"),
			UpdateTime: time.Now().Add(-9 * time.Minute).Format("2006-01-02 15:04:05"),
		},
		{
			RecordId:   "record_003",
			Role:       "user",
			Content:    "介绍一下go-zero框架",
			CreateTime: time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05"),
			UpdateTime: time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05"),
		},
		{
			RecordId:   "record_004",
			Role:       "assistant",
			Content:    "go-zero是一个集成了各种工程实践的web和rpc框架。通过弹性设计保障了大并发服务端的稳定性...",
			CreateTime: time.Now().Add(-4 * time.Minute).Format("2006-01-02 15:04:05"),
			UpdateTime: time.Now().Add(-4 * time.Minute).Format("2006-01-02 15:04:05"),
		},
	}

	// 根据分页参数返回数据
	start := (req.PageNumber - 1) * req.PageSize
	end := start + req.PageSize
	if start >= len(records) {
		start = len(records)
	}
	if end > len(records) {
		end = len(records)
	}

	paginatedRecords := records[start:end]
	if req.Sort == "desc" {
		// 倒序排列
		for i, j := 0, len(paginatedRecords)-1; i < j; i, j = i+1, j-1 {
			paginatedRecords[i], paginatedRecords[j] = paginatedRecords[j], paginatedRecords[i]
		}
	}

	resp = &types.GetChatRecordsResp{
		RecordList: paginatedRecords,
		Total:      len(records),
	}

	return resp, nil
}
