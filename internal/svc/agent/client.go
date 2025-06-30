// internal/svc/agent/client.go 修复版本
package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"ai-agent/core/eino/einoagent"      // 使用正确的项目路径
	espkg "ai-agent/core/elasticsearch" // 添加Elasticsearch包
	"ai-agent/core/memory"
)

type Config struct {
	ArkAPIKey         string
	ArkChatModel      string
	ArkEmbeddingModel string
	// 移除RedisAddr，添加Elasticsearch配置
	ElasticsearchAddr     string
	ElasticsearchUsername string
	ElasticsearchPassword string
	ElasticsearchIndex    string
	LogDir                string
	Debug                 bool
	// APMPlus 配置
	ApmplusAppKey string
	ApmplusRegion string
	// Langfuse 配置
	LangfusePublicKey string
	LangfuseSecretKey string
}

type Client struct {
	config    Config
	memory    *mem.SimpleMemory // 使用正确的内存管理
	cbHandler callbacks.Handler
	once      sync.Once
}

func NewClient(config Config) (*Client, error) {
	client := &Client{
		config: config,
		memory: mem.GetDefaultMemory(),
	}

	err := client.init()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) init() error {
	var err error
	c.once.Do(func() {
		// 首先初始化Elasticsearch
		err = c.initElasticsearch()
		if err != nil {
			return
		}

		err = c.initCallback()
		if err != nil {
			return
		}
		err = c.initGlobalCallbacks()
	})
	return err
}

// 添加Elasticsearch初始化函数
func (c *Client) initElasticsearch() error {
	// 设置环境变量（如果配置中有值）
	if c.config.ElasticsearchAddr != "" {
		os.Setenv("ELASTICSEARCH_ADDR", c.config.ElasticsearchAddr)
	}
	if c.config.ElasticsearchUsername != "" {
		os.Setenv("ELASTICSEARCH_USERNAME", c.config.ElasticsearchUsername)
	}
	if c.config.ElasticsearchPassword != "" {
		os.Setenv("ELASTICSEARCH_PASSWORD", c.config.ElasticsearchPassword)
	}
	if c.config.ElasticsearchIndex != "" {
		os.Setenv("ELASTICSEARCH_INDEX_NAME", c.config.ElasticsearchIndex)
	}

	// 初始化Elasticsearch客户端
	err := espkg.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize Elasticsearch: %w", err)
	}

	return nil
}

func (c *Client) initCallback() error {
	// 创建日志目录
	logDir := c.config.LogDir
	if logDir == "" {
		logDir = "log"
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// 打开日志文件
	logFile := fmt.Sprintf("%s/eino.log", logDir)
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// 创建回调配置
	cbConfig := &LogCallbackConfig{
		Detail: true,
		Writer: f,
		Debug:  c.config.Debug,
	}

	c.cbHandler = LogCallback(cbConfig)
	return nil
}

func (c *Client) initGlobalCallbacks() error {
	callbackHandlers := make([]callbacks.Handler, 0)

	// APMPlus 回调
	if c.config.ApmplusAppKey != "" {
		region := c.config.ApmplusRegion
		if region == "" {
			region = "cn-beijing"
		}

		cbh, _, err := apmplus.NewApmplusHandler(&apmplus.Config{
			Host:        fmt.Sprintf("apmplus-%s.volces.com:4317", region),
			AppKey:      c.config.ApmplusAppKey,
			ServiceName: "ai-agent-elasticsearch", // 更新服务名
			Release:     "release/v1.0.0",
		})
		if err != nil {
			return fmt.Errorf("failed to create apmplus handler: %w", err)
		}
		callbackHandlers = append(callbackHandlers, cbh)
	}

	// Langfuse 回调
	if c.config.LangfusePublicKey != "" && c.config.LangfuseSecretKey != "" {
		cbh, _ := langfuse.NewLangfuseHandler(&langfuse.Config{
			Host:      "https://cloud.langfuse.com",
			PublicKey: c.config.LangfusePublicKey,
			SecretKey: c.config.LangfuseSecretKey,
			Name:      "AI Agent with Elasticsearch",
			Public:    true,
			Release:   "release/v1.0.0",
			UserID:    "ai_agent_user",
			Tags:      []string{"ai-agent", "elasticsearch"},
		})
		callbackHandlers = append(callbackHandlers, cbh)
	}

	if len(callbackHandlers) > 0 {
		callbacks.InitCallbackHandlers(callbackHandlers)
	}

	return nil
}

// Stream 提供流式对话功能
func (c *Client) Stream(ctx context.Context, conversationID, message string) (*schema.StreamReader[*schema.Message], error) {
	// 构建 agent
	runner, err := einoagent.BuildEinoAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent graph: %w", err)
	}

	// 获取对话历史
	conversation := c.memory.GetConversation(conversationID, true)

	// 创建用户消息
	userMessage := &einoagent.UserMessage{
		ID:      conversationID,
		Query:   message,
		History: conversation.GetMessages(),
	}

	// 流式执行
	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(c.cbHandler))
	if err != nil {
		return nil, fmt.Errorf("failed to stream: %w", err)
	}

	srs := sr.Copy(2)

	go func() {
		// 保存到内存
		fullMsgs := make([]*schema.Message, 0)

		defer func() {
			srs[1].Close()

			// 添加用户输入到历史
			conversation.Append(schema.UserMessage(message))

			fullMsg, err := schema.ConcatMessages(fullMsgs)
			if err != nil {
				fmt.Printf("error concatenating messages: %v\n", err)
				return
			}
			conversation.Append(fullMsg)
		}()

		for {
			msg, err := srs[1].Recv()
			if err != nil {
				if err != io.EOF {
					fmt.Printf("error receiving message: %v\n", err)
				}
				break
			}
			fullMsgs = append(fullMsgs, msg)
		}
	}()

	return srs[0], nil
}
