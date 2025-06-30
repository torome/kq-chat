package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

	// Elasticsearch 配置 (替换Redis配置)
	ElasticsearchAddr     string `json:",env=ELASTICSEARCH_ADDR,default=http://localhost:9200"`
	ElasticsearchUsername string `json:",env=ELASTICSEARCH_USERNAME,optional"`
	ElasticsearchPassword string `json:",env=ELASTICSEARCH_PASSWORD,optional"`
	ElasticsearchIndex    string `json:",env=ELASTICSEARCH_INDEX_NAME,default=eino-knowledge-base"`

	// 知识库配置
	ThinkingEnabled         bool    `json:",env=THINKING_ENABLED,default=true"`         // 是否启用知识库
	KnowledgeEnabled        bool    `json:",env=KNOWLEDGE_ENABLED,default=true"`        // 是否启用知识库
	KnowledgeScoreThreshold float64 `json:",env=KNOWLEDGE_SCORE_THRESHOLD,default=0.5"` // 相似度阈值
	KnowledgeMaxResults     int     `json:",env=KNOWLEDGE_MAX_RESULTS,default=8"`       // 最大检索结果数

	// Agent 配置
	ArkAPIKey         string `json:",env=ARK_API_KEY"`
	ArkChatModel      string `json:",env=ARK_CHAT_MODEL"`
	ArkEmbeddingModel string `json:",env=ARK_EMBEDDING_MODEL"`
	RedisAddr         string `json:",default=49.235.180.215:6379"`
	LogDir            string `json:",default=log"`
	Debug             bool   `json:",env=DEBUG,default=false"`

	// 监控配置
	ApmplusAppKey     string `json:",env=APMPLUS_APP_KEY,optional"`
	ApmplusRegion     string `json:",env=APMPLUS_REGION,optional"`
	LangfusePublicKey string `json:",env=LANGFUSE_PUBLIC_KEY,optional"`
	LangfuseSecretKey string `json:",env=LANGFUSE_SECRET_KEY,optional"`
}
