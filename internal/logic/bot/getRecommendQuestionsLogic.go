package bot

import (
	"context"

	"ai-agent/internal/svc"
	"ai-agent/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRecommendQuestionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRecommendQuestionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRecommendQuestionsLogic {
	return &GetRecommendQuestionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRecommendQuestionsLogic) GetRecommendQuestions(req *types.GetRecommendQuestionsReq) error {
	// todo: add your logic here and delete this line

	return nil
}
