package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

	// Agent 配置
	ArkAPIKey         string `json:",env=ARK_API_KEY"`
	ArkChatModel      string `json:",env=ARK_CHAT_MODEL"`
	ArkEmbeddingModel string `json:",env=ARK_EMBEDDING_MODEL"`
	RedisAddr         string `json:",default=localhost:6379"`
	LogDir            string `json:",default=log"`
	Debug             bool   `json:",env=DEBUG,default=false"`

	// 监控配置
	ApmplusAppKey     string `json:",env=APMPLUS_APP_KEY,optional"`
	ApmplusRegion     string `json:",env=APMPLUS_REGION,optional"`
	LangfusePublicKey string `json:",env=LANGFUSE_PUBLIC_KEY,optional"`
	LangfuseSecretKey string `json:",env=LANGFUSE_SECRET_KEY,optional"`
}
