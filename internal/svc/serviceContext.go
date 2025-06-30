package svc

import (
	"ai-agent/internal/config"
	"ai-agent/internal/svc/agent"
	"log"
)

type ServiceContext struct {
	Config      config.Config
	AgentClient *agent.Client
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
	}
}
