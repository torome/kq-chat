package bot

import (
	"context"
	"fmt"
	"time"

	"ai-agent/internal/svc"
	"ai-agent/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TextToSpeechLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTextToSpeechLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TextToSpeechLogic {
	return &TextToSpeechLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TextToSpeechLogic) TextToSpeech(req *types.TextToSpeechReq) (resp *types.TextToSpeechResp, err error) {
	if req.TaskId != "" {
		// 查询任务状态
		resp = &types.TextToSpeechResp{
			Status:    2, // 2表示完成
			ResultUrl: "https://example.com/audio/" + req.TaskId + ".mp3",
		}
		return resp, nil
	}

	// 创建新任务
	taskId := fmt.Sprintf("task_%d", time.Now().UnixNano())
	l.Infof("TextToSpeech: botId=%s, text=%s, taskId=%s", req.BotId, req.Text, taskId)

	resp = &types.TextToSpeechResp{
		TaskId: taskId,
	}

	return resp, nil
}
