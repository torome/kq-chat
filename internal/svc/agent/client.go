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

	"ai-agent/core/eino/einoagent" // 替换为你的项目路径
	"ai-agent/core/memory"
)

type Config struct {
	ArkAPIKey         string
	ArkChatModel      string
	ArkEmbeddingModel string
	RedisAddr         string
	LogDir            string
	Debug             bool
	// APMPlus 配置
	ApmplusAppKey string
	ApmplusRegion string
	// Langfuse 配置
	LangfusePublicKey string
	LangfuseSecretKey string
}

type Client struct {
	config    Config
	memory    *mem.SimpleMemory // 直接使用具体类型
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
		err = c.initCallback()
		if err != nil {
			return
		}
		err = c.initGlobalCallbacks()
	})
	return err
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
			ServiceName: "kq-chat-agent",
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
			Name:      "KQ Chat Agent",
			Public:    true,
			Release:   "release/v1.0.0",
			UserID:    "kq_chat_user",
			Tags:      []string{"kq-chat", "agent"},
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

	// 开始流式处理
	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(c.cbHandler))
	if err != nil {
		return nil, fmt.Errorf("failed to stream: %w", err)
	}

	// 复制流以便保存历史记录
	srs := sr.Copy(2)

	// 异步保存对话历史
	go c.saveConversationHistory(ctx, srs[1], conversation, message)

	return srs[0], nil
}

func (c *Client) saveConversationHistory(ctx context.Context, sr *schema.StreamReader[*schema.Message], conversation *mem.Conversation, userMessage string) {
	fullMsgs := make([]*schema.Message, 0)

	defer func() {
		sr.Close()

		// 添加用户消息到历史
		conversation.Append(schema.UserMessage(userMessage))

		// 合并并添加 agent 响应到历史
		if len(fullMsgs) > 0 {
			fullMsg, err := schema.ConcatMessages(fullMsgs)
			if err != nil {
				fmt.Printf("error concatenating messages: %v\n", err)
				return
			}
			conversation.Append(fullMsg)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			chunk, err := sr.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				fmt.Printf("error receiving chunk: %v\n", err)
				return
			}
			fullMsgs = append(fullMsgs, chunk)
		}
	}
}

// GetConversationHistory 获取对话历史
func (c *Client) GetConversationHistory(conversationID string) *mem.Conversation {
	return c.memory.GetConversation(conversationID, false)
}

// DeleteConversation 删除对话
func (c *Client) DeleteConversation(conversationID string) error {
	return c.memory.DeleteConversation(conversationID)
}

// ListConversations 列出所有对话ID
func (c *Client) ListConversations() []string {
	return c.memory.ListConversations()
}
