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
	// 用于追踪待处理的工具调用
	var pendingToolCalls = make(map[string]*schema.ToolCall)
	var pendingToolCallsLock sync.Mutex

	return callbacks.NewHandlerBuilder().
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

				// 创建工具调用对象
				schemaToolCall := &schema.ToolCall{
					ID:   toolCallId,
					Type: "function",
					Function: schema.FunctionCall{
						Name:      info.Name,
						Arguments: string(inputStr),
					},
				}

				// 记录待处理的工具调用
				pendingToolCallsLock.Lock()
				pendingToolCalls[toolCallId] = schemaToolCall
				pendingToolCallsLock.Unlock()

				// 构造工具调用事件
				toolCallEvent := types.StreamEvent{
					Type:     "tool-call",
					RecordId: recordId,
					Role:     "assistant",
					ToolCall: schemaToolCall,
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

				fmt.Printf("Saved tool call with ID: %s\n", toolCallId)
			}
			return ctx
		}).
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
					ToolCallId: toolCallId,
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

				fmt.Printf("Tool call %s completed\n", toolCallId)
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if isToolCall(info) {
				fmt.Printf("=== Tool Call Error ===\n")
				fmt.Printf("Tool Name: %s, Error: %v\n", info.Name, err)

				// 获取工具调用ID并创建错误响应
				if toolCallId, ok := ctx.Value("toolCallId").(string); ok {
					toolResultMsg := &schema.Message{
						Role:       schema.Tool,
						Content:    fmt.Sprintf("Error: %v", err),
						ToolCallID: toolCallId,
						ToolName:   info.Name,
					}
					conversation.Append(toolResultMsg)
					fmt.Printf("Saved tool error message with ToolCallID: %s\n", toolCallId)
				}
			}
			return ctx
		}).
		Build()
}

// 修复StreamWithToolEvents方法，确保消息格式正确
func (c *Client) StreamWithToolEvents(ctx context.Context, conversationID, message string) (*StreamReaderWithToolEvents, error) {
	runner, err := einoagent.BuildEinoAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent graph: %w", err)
	}

	conversation := c.memory.GetConversation(conversationID, true)

	// 获取历史消息并进行格式验证和修复
	historyMessages := conversation.GetMessages()

	// 验证和修复消息格式
	if err := c.validateAndFixMessages(historyMessages); err != nil {
		fmt.Printf("Warning: message validation failed: %v\n", err)
		// 可以选择清理有问题的消息
		historyMessages = c.cleanInvalidMessages(historyMessages)
	}

	userMessage := &einoagent.UserMessage{
		ID:      conversationID,
		Query:   message,
		History: historyMessages,
	}

	// 创建工具事件通道
	toolEvents := make(chan ToolEvent, 10)
	streamCtx, cancel := context.WithCancel(ctx)

	// 创建自定义的消息处理回调
	messageHandler := c.createMessageHandlerCallback(conversation, toolEvents, conversationID)

	// 流式执行
	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(c.cbHandler, messageHandler))
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

			// 添加用户消息
			conversation.Append(schema.UserMessage(message))

			// 处理完整响应
			fullMsg, err := schema.ConcatMessages(fullMsgs)
			if err != nil {
				fmt.Printf("error concatenating messages: %v\n", err)
				return
			}

			// 确保消息格式正确后再保存
			if c.validateSingleMessage(fullMsg) == nil {
				conversation.Append(fullMsg)
			} else {
				fmt.Printf("Warning: Invalid message format, not saving to conversation\n")
			}
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

// 创建消息处理回调，确保工具调用和响应的配对
func (c *Client) createMessageHandlerCallback(conversation *mem.Conversation, toolEvents chan ToolEvent, recordId string) callbacks.Handler {
	// 追踪工具调用状态
	toolCallTracker := &ToolCallTracker{
		pendingCalls: make(map[string]*PendingToolCall),
		mutex:        sync.Mutex{},
	}

	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if isToolCall(info) {
				return toolCallTracker.handleToolStart(ctx, info, input, toolEvents, recordId)
			}
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if isToolCall(info) {
				return toolCallTracker.handleToolEnd(ctx, info, output, conversation, toolEvents, recordId)
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if isToolCall(info) {
				return toolCallTracker.handleToolError(ctx, info, err, conversation, toolEvents, recordId)
			}
			return ctx
		}).
		Build()
}

// 工具调用追踪器
type ToolCallTracker struct {
	pendingCalls map[string]*PendingToolCall
	mutex        sync.Mutex
}

type PendingToolCall struct {
	ID        string
	Name      string
	Arguments string
	StartTime time.Time
	ToolCall  *schema.ToolCall
}

func (t *ToolCallTracker) handleToolStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput, toolEvents chan ToolEvent, recordId string) context.Context {
	toolCallId := fmt.Sprintf("call_%s_%d", info.Name, time.Now().UnixNano())
	ctx = context.WithValue(ctx, "toolCallId", toolCallId)

	inputStr, _ := json.Marshal(convertInputToMap(input))

	schemaToolCall := &schema.ToolCall{
		ID:   toolCallId,
		Type: "function",
		Function: schema.FunctionCall{
			Name:      info.Name,
			Arguments: string(inputStr),
		},
	}

	// 记录待处理的工具调用
	t.mutex.Lock()
	t.pendingCalls[toolCallId] = &PendingToolCall{
		ID:        toolCallId,
		Name:      info.Name,
		Arguments: string(inputStr),
		StartTime: time.Now(),
		ToolCall:  schemaToolCall,
	}
	t.mutex.Unlock()

	// 发送工具调用事件
	toolCallEvent := types.StreamEvent{
		Type:     "tool-call",
		RecordId: recordId,
		Role:     "assistant",
		ToolCall: schemaToolCall,
	}

	select {
	case toolEvents <- ToolEvent{
		Type:     "tool-call",
		Event:    toolCallEvent,
		RecordId: recordId,
	}:
		fmt.Printf("Sent tool-call event: %s\n", toolCallId)
	default:
		fmt.Printf("Tool events channel is full\n")
	}

	return ctx
}

func (t *ToolCallTracker) handleToolEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput, conversation *mem.Conversation, toolEvents chan ToolEvent, recordId string) context.Context {
	toolCallId, ok := ctx.Value("toolCallId").(string)
	if !ok {
		fmt.Printf("Warning: tool_call_id not found in context\n")
		return ctx
	}

	t.mutex.Lock()
	pendingCall, exists := t.pendingCalls[toolCallId]
	if exists {
		delete(t.pendingCalls, toolCallId)
	}
	t.mutex.Unlock()

	if !exists {
		fmt.Printf("Warning: no pending call found for %s\n", toolCallId)
		return ctx
	}

	// 构造并保存工具调用消息（如果还没保存）
	toolCallMsg := &schema.Message{
		Role:      schema.Assistant,
		Content:   "",
		ToolCalls: []schema.ToolCall{*pendingCall.ToolCall},
	}
	conversation.Append(toolCallMsg)

	// 构造工具响应消息
	var outputStr string
	if outputBytes, err := json.Marshal(output); err == nil {
		outputStr = string(outputBytes)
	} else {
		outputStr = fmt.Sprintf("%v", output)
	}

	toolResultMsg := &schema.Message{
		Role:       schema.Tool,
		Content:    outputStr,
		ToolCallID: toolCallId, // 关键：设置ToolCallID
		ToolName:   info.Name,
	}
	conversation.Append(toolResultMsg)

	// 发送工具结果事件
	toolResultEvent := types.StreamEvent{
		Type:       "tool-result",
		RecordId:   recordId,
		Role:       "assistant",
		ToolCallId: toolCallId,
		Result:     convertOutputToMap(output),
	}

	select {
	case toolEvents <- ToolEvent{
		Type:     "tool-result",
		Event:    toolResultEvent,
		RecordId: recordId,
	}:
		fmt.Printf("Sent tool-result event: %s\n", toolCallId)
	default:
		fmt.Printf("Tool events channel is full\n")
	}

	fmt.Printf("Tool call %s completed and saved\n", toolCallId)
	return ctx
}

func (t *ToolCallTracker) handleToolError(ctx context.Context, info *callbacks.RunInfo, err error, conversation *mem.Conversation, toolEvents chan ToolEvent, recordId string) context.Context {
	toolCallId, ok := ctx.Value("toolCallId").(string)
	if !ok {
		return ctx
	}

	t.mutex.Lock()
	pendingCall, exists := t.pendingCalls[toolCallId]
	if exists {
		delete(t.pendingCalls, toolCallId)
	}
	t.mutex.Unlock()

	if exists {
		// 保存工具调用消息
		toolCallMsg := &schema.Message{
			Role:      schema.Assistant,
			Content:   "",
			ToolCalls: []schema.ToolCall{*pendingCall.ToolCall},
		}
		conversation.Append(toolCallMsg)

		// 保存错误响应
		toolResultMsg := &schema.Message{
			Role:       schema.Tool,
			Content:    fmt.Sprintf("Error: %v", err),
			ToolCallID: toolCallId,
			ToolName:   info.Name,
		}
		conversation.Append(toolResultMsg)
	}

	return ctx
}

// 消息验证和修复函数
func (c *Client) validateAndFixMessages(messages []*schema.Message) error {
	var errors []string

	for i, msg := range messages {
		if err := c.validateSingleMessage(msg); err != nil {
			errors = append(errors, fmt.Sprintf("message %d: %v", i, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %v", errors)
	}
	return nil
}

func (c *Client) validateSingleMessage(msg *schema.Message) error {
	if msg.Role == schema.Tool {
		if msg.ToolCallID == "" {
			return fmt.Errorf("tool message missing ToolCallID")
		}
	}

	if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
		for j, toolCall := range msg.ToolCalls {
			if toolCall.ID == "" {
				return fmt.Errorf("tool_call %d missing ID", j)
			}
		}
	}

	return nil
}

func (c *Client) cleanInvalidMessages(messages []*schema.Message) []*schema.Message {
	cleanMessages := make([]*schema.Message, 0, len(messages))

	for _, msg := range messages {
		if c.validateSingleMessage(msg) == nil {
			cleanMessages = append(cleanMessages, msg)
		} else {
			fmt.Printf("Removing invalid message: %+v\n", msg)
		}
	}

	return cleanMessages
}

// 辅助函数保持不变
func isToolCall(info *callbacks.RunInfo) bool {
	toolTypes := map[string]bool{
		"TaskManager":        true,
		"task_manager":       true,
		"OpenFileTool":       false,
		"open":               false,
		"GitCloneFile":       false,
		"gitclone":           false,
		"EinoAssistant":      false,
		"eino_tool":          true,
		"DuckDuckGo":         false,
		"duckduckgo":         false,
		"calculator":         true,
		"vocabulary_manager": true,
	}

	componentStr := string(info.Component)
	typeStr := string(info.Type)
	return toolTypes[componentStr] || toolTypes[typeStr] || toolTypes[info.Name]
}

func convertInputToMap(input callbacks.CallbackInput) map[string]interface{} {
	if input == nil {
		return map[string]interface{}{}
	}

	if inputMap, ok := input.(map[string]interface{}); ok {
		return inputMap
	}

	// 尝试JSON序列化再反序列化
	if inputBytes, err := json.Marshal(input); err == nil {
		var result map[string]interface{}
		if json.Unmarshal(inputBytes, &result) == nil {
			return result
		}
	}

	return map[string]interface{}{"input": fmt.Sprintf("%v", input)}
}

func convertOutputToMap(output callbacks.CallbackOutput) map[string]interface{} {
	if output == nil {
		return map[string]interface{}{}
	}

	if outputMap, ok := output.(map[string]interface{}); ok {
		return outputMap
	}

	if outputBytes, err := json.Marshal(output); err == nil {
		var result map[string]interface{}
		if json.Unmarshal(outputBytes, &result) == nil {
			return result
		}
	}

	return map[string]interface{}{"output": fmt.Sprintf("%v", output)}
}
