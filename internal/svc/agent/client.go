// internal/svc/agent/client.go 修改版本 - 添加工具事件传递
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"ai-agent/core/eino/einoagent"
	espkg "ai-agent/core/elasticsearch"
	"ai-agent/core/memory"
	"ai-agent/internal/types"
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

// StreamWithToolEvents 提供带工具事件的流式对话功能
func (c *Client) StreamWithToolEvents(ctx context.Context, conversationID, message string) (*StreamReaderWithToolEvents, error) {
	runner, err := einoagent.BuildEinoAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent graph: %w", err)
	}

	conversation := c.memory.GetConversation(conversationID, true)

	userMessage := &einoagent.UserMessage{
		ID:      conversationID,
		Query:   message,
		History: conversation.GetMessages(),
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
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			// 判断是否是工具调用（检查组件类型）
			if isToolCall(info) {
				fmt.Printf("=== Tool Call Started ===\n")
				fmt.Printf("Tool Name: %s, Type: %s, Component: %s\n", info.Name, info.Type, info.Component)
				fmt.Printf("Tool Input: %+v\n", input)

				// 生成唯一的工具调用ID
				toolCallId := fmt.Sprintf("call_%s_%d", info.Name, time.Now().UnixNano())

				// 将工具调用ID存储到context中，供OnEnd使用
				ctx = context.WithValue(ctx, "toolCallId", toolCallId)

				// 构造工具调用事件
				toolCallEvent := types.StreamEvent{
					Type:     "tool-call",
					RecordId: recordId,
					Role:     "assistant",
					ToolCall: &types.ToolCall{
						Id:   toolCallId,
						Type: "function",
						Function: types.ToolCallFunction{
							Name:      info.Name,
							Arguments: convertInputToMap(input),
						},
					},
				}

				// 立即发送工具调用事件
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

				// 保存工具调用信息到对话历史
				inputStr, _ := json.Marshal(input)
				toolCallMsg := &schema.Message{
					Role:    schema.Assistant,
					Content: fmt.Sprintf("调用工具: %s，参数: %s", info.Name, string(inputStr)),
				}
				conversation.Append(toolCallMsg)
			}
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			// 判断是否是工具调用完成
			if isToolCall(info) {
				fmt.Printf("=== Tool Call Ended ===\n")
				fmt.Printf("Tool Name: %s, Type: %s, Component: %s\n", info.Name, info.Type, info.Component)
				fmt.Printf("Tool Output: %+v\n", output)

				// 从context中获取工具调用ID
				toolCallId, ok := ctx.Value("toolCallId").(string)
				if !ok {
					toolCallId = fmt.Sprintf("call_%s_%d", info.Name, time.Now().UnixNano())
				}

				// 构造工具调用结果事件，添加callResult字段
				toolResultEvent := types.StreamEvent{
					Type:       "tool-result",
					RecordId:   recordId,
					Role:       "assistant",
					ToolCallId: toolCallId,
					Result:     convertOutputToMap(output),
				}

				// 立即发送工具调用结果事件
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

				// 保存工具返回结果到对话历史
				var outputStr string
				if outputBytes, err := json.Marshal(output); err == nil {
					outputStr = string(outputBytes)
				} else {
					outputStr = fmt.Sprintf("%v", output)
				}

				toolResultMsg := &schema.Message{
					Role:    schema.Tool,
					Content: outputStr,
				}
				conversation.Append(toolResultMsg)
				fmt.Printf("Saved tool result message: %s\n", outputStr)
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
		"TaskManager":   true,
		"task_manager":  true,
		"OpenFileTool":  true,
		"open":          true,
		"GitCloneFile":  true,
		"gitclone":      true,
		"EinoAssistant": true,
		"eino_tool":     true,
		"DuckDuckGo":    true,
		"duckduckgo":    true,
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
