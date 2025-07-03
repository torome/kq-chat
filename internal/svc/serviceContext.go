package svc

import (
	"ai-agent/internal/config"
	"ai-agent/internal/svc/agent"
	"ai-agent/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"log"
)

type ServiceContext struct {
	Config      config.Config
	AgentClient *agent.Client

	CircleModel model.CircleModel
	TaskModel   model.TaskModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化 agent 客户端
	agentClient, err := agent.NewClient(agent.Config{
		ArkAPIKey:         c.ArkAPIKey,
		ArkChatModel:      c.ArkChatModel,
		ArkEmbeddingModel: c.ArkEmbeddingModel,
		//RedisAddr:         c.RedisAddr,
		LogDir:            c.LogDir,
		Debug:             c.Debug,
		ApmplusAppKey:     c.ApmplusAppKey,
		ApmplusRegion:     c.ApmplusRegion,
		LangfusePublicKey: c.LangfusePublicKey,
		LangfuseSecretKey: c.LangfuseSecretKey,
	})
	if err != nil {
		log.Fatalf("Failed to initialize agent client: %v", err)
	}

	return &ServiceContext{
		Config:      c,
		AgentClient: agentClient,

		CircleModel: model.NewCircleModel(sqlx.NewMysql(c.DB.DataSource), c.Cache),
		TaskModel:   model.NewTaskModel(sqlx.NewMysql(c.DB.DataSource), c.Cache),
	}
}
