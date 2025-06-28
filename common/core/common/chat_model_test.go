package common

import (
	"fmt"
	"github.com/cloudwego/eino/schema"
	"testing"

	"github.com/gogf/gf/v2/os/gctx"
)

func TestCfg(t *testing.T) {
	ctx := gctx.New()

	cm, err := GetImageModel(ctx, nil)
	if err != nil {
		return
	}
	generate, err := cm.Generate(ctx, []*schema.Message{
		{
			Role: schema.System,
			Content: fmt.Sprintf("你是一个专业的问题生成助手，任务是从给定的文本中提取或生成可能的问题。你不需要回答这些问题，只需生成问题本身。\n"+
				"知识库名字是：《%s》\n\n"+
				"输出格式：\n"+
				"- 每个问题占一行\n"+
				"- 问题必须以问号结尾\n"+
				"- 避免重复或语义相似的问题\n\n"+
				"生成规则：\n"+
				"- 生成的问题必须严格基于文本内容，不能脱离文本虚构。\n"+
				"- 优先生成事实性问题（如谁、何时、何地、如何）。\n"+
				"- 对于复杂文本，可生成多层次问题（基础事实 + 推理问题）。\n"+
				"- 禁止生成主观或开放式问题（如“你认为...？”）。"+
				"- 数量控制在3-5个", knowledgeName),
		},
		{
			Role:    schema.User,
			Content: "",
		},
	})
	if err != nil {
		return
	}
	qaContent := generate.Content
}
