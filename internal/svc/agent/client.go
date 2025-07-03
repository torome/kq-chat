// internal/svc/agent/client.go 修改版本 - 添加工具事件传递
package agent

import (
	"ai-agent/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"io"
	"os"
	"sync"
	"time"

	"ai-agent/core/eino/einoagent"
	espkg "ai-agent/core/elasticsearch"
	"ai-agent/core/memory"
)

type Config struct {
	ArkAPIKey             string
	ArkChatModel          string
	ArkEmbeddingModel     string
	ElasticsearchAddr     string
	ElasticsearchUsername string
	ElasticsearchPassword string
	ElasticsearchIndex    string
	LogDir                string
	Debug                 bool
	ApmplusAppKey         string
	ApmplusRegion         string
	LangfusePublicKey     string
	LangfuseSecretKey     string
}

// ToolEvent 工具事件结构
type ToolEvent struct {
	Type     string      `json:"type"`  // "tool-call" | "tool-result"
	Event    interface{} `json:"event"` // 具体的事件数据
	RecordId string      `json:"record_id"`
}

// StreamReaderWithToolEvents 包装流读取器和工具事件通道
type StreamReaderWithToolEvents struct {
	reader     *schema.StreamReader[*schema.Message]
	toolEvents chan ToolEvent
	ctx        context.Context
	cancel     context.CancelFunc
}

func (s *StreamReaderWithToolEvents) Recv() (*schema.Message, error) {
	return s.reader.Recv()
}

func (s *StreamReaderWithToolEvents) Close() error {
	s.cancel()
	close(s.toolEvents)
	s.reader.Close()
	return nil
}

func (s *StreamReaderWithToolEvents) ToolEvents() <-chan ToolEvent {
	return s.toolEvents
}

type Client struct {
	config    Config
	memory    *mem.SimpleMemory
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

func (c *Client) initElasticsearch() error {
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

	err := espkg.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize Elasticsearch: %w", err)
	}

	return nil
}

func (c *Client) initCallback() error {
	logDir := c.config.LogDir
	if logDir == "" {
		logDir = "log"
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile := fmt.Sprintf("%s/eino.log", logDir)
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

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

	if c.config.ApmplusAppKey != "" {
		region := c.config.ApmplusRegion
		if region == "" {
			region = "cn-beijing"
		}

		cbh, _, err := apmplus.NewApmplusHandler(&apmplus.Config{
			Host:        fmt.Sprintf("apmplus-%s.volces.com:4317", region),
			AppKey:      c.config.ApmplusAppKey,
			ServiceName: "ai-agent-elasticsearch",
			Release:     "release/v1.0.0",
		})
		if err != nil {
			return fmt.Errorf("failed to create apmplus handler: %w", err)
		}
		callbackHandlers = append(callbackHandlers, cbh)
	}

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

// 额外的调试和验证函数
func validateConversationMessages(messages []*schema.Message) error {
	for i, msg := range messages {
		if msg.Role == schema.Tool {
			if msg.ToolCallID == "" {
				return fmt.Errorf("message %d: tool message missing ToolCallID", i)
			}
		}
		if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			for j, toolCall := range msg.ToolCalls {
				if toolCall.ID == "" {
					return fmt.Errorf("message %d, tool_call %d: missing ID", i, j)
				}
			}
		}
	}
	return nil
}

// 在 StreamWithToolEvents 方法中添加验证
func (c *Client) StreamWithToolEvents(ctx context.Context, conversationID, message string) (*StreamReaderWithToolEvents, error) {
	runner, err := einoagent.BuildEinoAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent graph: %w", err)
	}

	conversation := c.memory.GetConversation(conversationID, true)

	// 获取历史消息并验证格式
	historyMessages := conversation.GetMessages()

	// 添加消息格式验证（可选，用于调试）
	if err := validateConversationMessages(historyMessages); err != nil {
		fmt.Printf("Warning: conversation message validation failed: %v\n", err)
		// 可以选择清理有问题的消息或继续执行
	}

	userMessage := &einoagent.UserMessage{
		ID:      conversationID,
		Query:   message,
		History: historyMessages,
	}

	// 创建工具事件通道
	toolEvents := make(chan ToolEvent, 10)
	streamCtx, cancel := context.WithCancel(ctx)

	// 创建工具回调处理器
	toolCallCapture := createToolCallbackHandlerWithEvents(conversation, toolEvents, conversationID)

	// 流式执行
	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(c.cbHandler, toolCallCapture))
	if err != nil {
		cancel()
		close(toolEvents)
		return nil, fmt.Errorf("failed to stream: %w", err)
	}

	srs := sr.Copy(2)

	go func() {
		fullMsgs := make([]*schema.Message, 0)

		defer func() {
			srs[1].Close()
			cancel()

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

	return &StreamReaderWithToolEvents{
		reader:     srs[0],
		toolEvents: toolEvents,
		ctx:        streamCtx,
		cancel:     cancel,
	}, nil
}

// 清理历史对话数据的辅助函数（可选）
func (c *Client) CleanConversationHistory(conversationID string) error {
	conversation := c.memory.GetConversation(conversationID, false)
	if conversation == nil {
		return nil
	}

	// 清理有问题的消息
	messages := conversation.GetMessages()
	cleanMessages := make([]*schema.Message, 0, len(messages))

	for _, msg := range messages {
		// 跳过格式有问题的工具消息
		if msg.Role == schema.Tool && msg.ToolCallID == "" {
			fmt.Printf("Skipping invalid tool message without ToolCallID\n")
			continue
		}
		cleanMessages = append(cleanMessages, msg)
	}

	// 重置对话历史（这里需要根据你的内存实现来调整）
	// conversation.SetMessages(cleanMessages)

	return nil
}

// Stream 保持向后兼容性
func (c *Client) Stream(ctx context.Context, conversationID, message string) (*schema.StreamReader[*schema.Message], error) {
	wrapper, err := c.StreamWithToolEvents(ctx, conversationID, message)
	if err != nil {
		return nil, err
	}

	// 启动goroutine来消费工具事件，避免阻塞
	go func() {
		for range wrapper.ToolEvents() {
			// 消费但不处理，保持向后兼容
		}
	}()

	return wrapper.reader, nil
}

// createToolCallbackHandlerWithEvents 创建带事件发送的工具回调处理器
func createToolCallbackHandlerWithEvents(conversation *mem.Conversation, toolEvents chan ToolEvent, recordId string) callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		// 在 OnStartFn 回调中，修复工具调用消息的保存
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if isToolCall(info) {
				fmt.Printf("=== Tool Call Started ===\n")
				fmt.Printf("Tool Name: %s, Type: %s, Component: %s\n", info.Name, info.Type, info.Component)
				fmt.Printf("Tool Input: %+v\n", input)

				// 生成唯一的工具调用ID
				toolCallId := fmt.Sprintf("call_%s_%d", info.Name, time.Now().UnixNano())
				ctx = context.WithValue(ctx, "toolCallId", toolCallId)

				// 将输入参数转换为 JSON 字符串
				inputStr, _ := json.Marshal(convertInputToMap(input))

				// 直接使用 schema.ToolCall
				schemaToolCall := &schema.ToolCall{
					ID:   toolCallId,
					Type: "function",
					Function: schema.FunctionCall{ // 注意：值类型，不是指针
						Name:      info.Name,
						Arguments: string(inputStr),
					},
				}

				// 构造工具调用事件
				toolCallEvent := types.StreamEvent{
					Type:     "tool-call",
					RecordId: recordId,
					Role:     "assistant",
					ToolCall: schemaToolCall, // 直接使用 schema.ToolCall
				}

				// 发送工具调用事件
				select {
				case toolEvents <- ToolEvent{
					Type:     "tool-call",
					Event:    toolCallEvent,
					RecordId: recordId,
				}:
					fmt.Printf("Sent tool-call event: %s\n", toolCallId)
				default:
					fmt.Printf("Tool events channel is full, skipping tool-call event\n")
				}

				// 保存工具调用消息到对话历史
				toolCallMsg := &schema.Message{
					Role:      schema.Assistant,
					Content:   "",                                 // 工具调用消息内容为空
					ToolCalls: []schema.ToolCall{*schemaToolCall}, // 使用相同的 schema.ToolCall
				}
				conversation.Append(toolCallMsg)
				fmt.Printf("Saved tool call message with ID: %s\n", toolCallId)
			}
			return ctx
		}).

		// 在 OnEndFn 回调中：
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if isToolCall(info) {
				fmt.Printf("=== Tool Call Ended ===\n")
				fmt.Printf("Tool Name: %s, Type: %s, Component: %s\n", info.Name, info.Type, info.Component)
				fmt.Printf("Tool Output: %+v\n", output)

				// 从context中获取工具调用ID
				toolCallId, ok := ctx.Value("toolCallId").(string)
				if !ok {
					toolCallId = fmt.Sprintf("call_%s_%d", info.Name, time.Now().UnixNano())
					fmt.Printf("Warning: tool_call_id not found in context, generated new one: %s\n", toolCallId)
				}

				// 构造工具调用结果事件
				toolResultEvent := types.StreamEvent{
					Type:       "tool-result",
					RecordId:   recordId,
					Role:       "assistant",
					ToolCallId: toolCallId, // 这个字段现在是 tool_call_id 格式
					Result:     convertOutputToMap(output),
				}

				// 发送工具调用结果事件
				select {
				case toolEvents <- ToolEvent{
					Type:     "tool-result",
					Event:    toolResultEvent,
					RecordId: recordId,
				}:
					fmt.Printf("Sent tool-result event: %s\n", toolCallId)
				default:
					fmt.Printf("Tool events channel is full, skipping tool-result event\n")
				}

				// 关键修复：正确保存工具返回结果到对话历史
				var outputStr string
				if outputBytes, err := json.Marshal(output); err == nil {
					outputStr = string(outputBytes)
				} else {
					outputStr = fmt.Sprintf("%v", output)
				}

				// 使用正确的 ToolCallID 字段设置
				toolResultMsg := &schema.Message{
					Role:       schema.Tool,
					Content:    outputStr,
					ToolCallID: toolCallId, // 关键修复：设置 ToolCallID 字段
					ToolName:   info.Name,  // 可选：设置工具名称
				}

				conversation.Append(toolResultMsg)
				fmt.Printf("Saved tool result message with ToolCallID: %s\n", toolCallId)
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if isToolCall(info) {
				fmt.Printf("=== Tool Call Error ===\n")
				fmt.Printf("Tool Name: %s, Error: %v\n", info.Name, err)
			}
			return ctx
		}).
		Build()
}

// isToolCall 判断是否是工具调用
func isToolCall(info *callbacks.RunInfo) bool {
	// 根据你的工具类型进行判断
	toolTypes := map[string]bool{
		"TaskManager":        true,
		"task_manager":       true,
		"OpenFileTool":       true,
		"open":               true,
		"GitCloneFile":       true,
		"gitclone":           true,
		"EinoAssistant":      true,
		"eino_tool":          true,
		"DuckDuckGo":         true,
		"duckduckgo":         true,
		"calculator":         true,
		"vocabulary_manager": true,
	}

	// 检查组件名称或类型，需要转换为字符串
	componentStr := string(info.Component)
	typeStr := string(info.Type)

	return toolTypes[componentStr] || toolTypes[typeStr] || toolTypes[info.Name]
}

// convertInputToMap 将输入转换为map
func convertInputToMap(input callbacks.CallbackInput) map[string]interface{} {
	if input == nil {
		return map[string]interface{}{}
	}

	// 尝试直接转换
	if inputMap, ok := input.(map[string]interface{}); ok {
		return inputMap
	}

	// 通过JSON序列化/反序列化转换
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return map[string]interface{}{"raw": fmt.Sprintf("%v", input)}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(inputBytes, &result); err != nil {
		return map[string]interface{}{"raw": string(inputBytes)}
	}

	return result
}

// convertOutputToMap 将输出转换为map
func convertOutputToMap(output callbacks.CallbackOutput) map[string]interface{} {
	if output == nil {
		return map[string]interface{}{}
	}

	// 尝试直接转换
	if outputMap, ok := output.(map[string]interface{}); ok {
		return outputMap
	}

	// 通过JSON序列化/反序列化转换
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return map[string]interface{}{"raw": fmt.Sprintf("%v", output)}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(outputBytes, &result); err != nil {
		return map[string]interface{}{"raw": string(outputBytes)}
	}

	return result
}
