package websocket

import (
	"ai-agent/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"time"

	"ai-agent/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type WebsocketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	conn   *websocket.Conn
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// WebSocket 消息类型
type WSMessage struct {
	Type string      `json:"type"` // sendMessage, getRecommendQuestions
	Data interface{} `json:"data"`
}

func NewWebsocketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WebsocketLogic {
	return &WebsocketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *WebsocketLogic) Websocket(w http.ResponseWriter, r *http.Request) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.Errorf("WebSocket upgrade failed: %v", err)
		return err
	}
	defer conn.Close()

	l.conn = conn
	l.Info("WebSocket connection established")

	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			l.Errorf("Read message failed: %v", err)
			break
		}

		switch msg.Type {
		case "sendMessage":
			l.handleSendMessage(msg.Data)
		case "getRecommendQuestions":
			l.handleRecommendQuestions(msg.Data)
		default:
			l.Errorf("Unknown message type: %s", msg.Type)
		}
	}

	return nil
}

// 处理发送消息
func (l *WebsocketLogic) handleSendMessage(data interface{}) {
	dataBytes, _ := json.Marshal(data)
	var sendData types.SendMessageReq
	json.Unmarshal(dataBytes, &sendData)

	l.Infof("Handling sendMessage: %+v", sendData)

	// 生成记录ID
	recordId := fmt.Sprintf("record_%d", time.Now().UnixNano())

	// 模拟流式响应
	l.simulateStreamResponse(sendData, recordId)
}

// 模拟流式响应
func (l *WebsocketLogic) simulateStreamResponse(data types.SendMessageReq, recordId string) {
	var events []types.StreamEvent

	// 1. 如果启用搜索，先发送搜索事件
	if data.SearchEnable {
		searchEvent := types.StreamEvent{
			Type:     "search",
			RecordId: recordId,
			Role:     "assistant",
			SearchInfo: &types.SearchInfo{
				SearchResults: []types.SearchResult{
					{Title: "Go语言官方文档", Url: "https://golang.org/doc/"},
					{Title: "go-zero框架介绍", Url: "https://go-zero.dev/"},
					{Title: "WebSocket实现指南", Url: "https://example.com/websocket-guide"},
				},
			},
		}
		events = append(events, searchEvent)
	}

	// 2. 思考过程
	thinkingContents := []string{
		"正在分析用户的问题...",
		"搜索相关的知识库内容...",
		"整理思路，准备回答...",
		"检查答案的准确性...",
	}

	for _, thinking := range thinkingContents {
		thinkingEvent := types.StreamEvent{
			Type:             "thinking",
			ReasoningContent: thinking,
			RecordId:         recordId,
			Role:             "assistant",
		}
		events = append(events, thinkingEvent)
	}

	// 3. 工具调用示例
	if strings.Contains(data.Msg, "查询天气") || strings.Contains(data.Msg, "搜索信息") || strings.Contains(data.Msg, "工具") {
		// 工具调用请求
		toolCallEvent := types.StreamEvent{
			Type:     "tool-call",
			RecordId: recordId,
			Role:     "assistant",
			ToolCall: &types.ToolCall{
				Id:   "call_weather_001",
				Type: "function",
				Function: types.ToolCallFunction{
					Name: "weather/getCurrentWeather",
					Arguments: map[string]interface{}{
						"location": "北京",
						"unit":     "celsius",
					},
				},
			},
		}
		events = append(events, toolCallEvent)

		// 工具调用响应
		toolResultEvent := types.StreamEvent{
			Type:       "tool-result",
			RecordId:   recordId,
			Role:       "assistant",
			ToolCallId: "call_weather_001",
			Result: map[string]interface{}{
				"location":    "北京",
				"temperature": "22°C",
				"weather":     "晴天",
				"humidity":    "65%",
			},
		}
		events = append(events, toolResultEvent)

		// 工具调用内容输出
		toolContentEvent := types.StreamEvent{
			Type:     "text",
			Content:  "根据天气查询工具的结果，我为您查询到以下信息：\n\n北京当前天气情况：\n- 温度：22°C\n- 天气：晴天\n- 湿度：65%\n\n建议您今天适合外出活动。",
			RecordId: recordId,
			Role:     "assistant",
		}
		events = append(events, toolContentEvent)
	} else {
		// 4. 普通文本回答
		responseText := l.generateResponse(data.Msg)
		textParts := l.splitTextForStreaming(responseText)

		for _, part := range textParts {
			textEvent := types.StreamEvent{
				Type:     "text",
				Content:  part,
				RecordId: recordId,
				Role:     "assistant",
			}
			events = append(events, textEvent)
		}
	}

	// 5. 结束事件
	finishEvent := types.StreamEvent{
		Type:         "finish",
		RecordId:     recordId,
		Role:         "assistant",
		FinishReason: "stop",
	}
	events = append(events, finishEvent)

	// 发送所有事件
	for i, event := range events {
		// 模拟实际流式输出的延迟
		if i > 0 {
			time.Sleep(200 * time.Millisecond)
		}

		// 发送事件
		eventData := fmt.Sprintf("data: %s\n\n", l.marshalEvent(event))
		if err := l.conn.WriteMessage(websocket.TextMessage, []byte(eventData)); err != nil {
			l.Errorf("Write message failed: %v", err)
			return
		}
	}

	// 发送结束标志
	endData := "data: [DONE]\n\n"
	l.conn.WriteMessage(websocket.TextMessage, []byte(endData))
}

// 处理推荐问题
func (l *WebsocketLogic) handleRecommendQuestions(data interface{}) {
	dataBytes, _ := json.Marshal(data)
	var recommendData types.GetRecommendQuestionsReq
	json.Unmarshal(dataBytes, &recommendData)

	l.Infof("Handling getRecommendQuestions: %+v", recommendData)

	// 模拟推荐问题的流式输出
	questions := []string{
		"您还想了解什么其他信息？",
		"需要我为您详细解释某个概念吗？",
		"有什么具体的问题需要深入探讨？",
	}

	for _, question := range questions {
		time.Sleep(100 * time.Millisecond)
		if err := l.conn.WriteMessage(websocket.TextMessage, []byte(question+"\n")); err != nil {
			l.Errorf("Write recommend question failed: %v", err)
			return
		}
	}
}

// 生成回答内容
func (l *WebsocketLogic) generateResponse(msg string) string {
	responses := map[string]string{
		"你好":          "您好！我是AI助手，很高兴为您服务。有什么我可以帮助您的吗？",
		"介绍一下go-zero": "go-zero是一个集成了各种工程实践的web和rpc框架。通过弹性设计保障了大并发服务端的稳定性，经受了充分的实战检验。\n\ngo-zero的主要特性包括：\n1. 强大的工具支持，尽可能少的代码编写\n2. 极简的接口\n3. 完全兼容net/http\n4. 支持中间件，方便扩展\n5. 高性能\n6. 面向故障编程，弹性设计\n7. 内建服务发现、负载均衡\n8. 内建限流、熔断、降级，且自动触发，自动恢复\n9. API参数自动校验\n10. 超时级联控制",
		"什么是微服务":      "微服务是一种软件架构风格，它将应用程序构建为一组小型、松散耦合的服务。每个服务都围绕特定的业务功能构建，可以独立开发、部署和扩展。\n\n微服务的主要特点：\n1. 单一职责：每个服务专注于一个业务领域\n2. 独立部署：服务可以独立发布和升级\n3. 技术栈自由：不同服务可以使用不同的技术\n4. 故障隔离：一个服务的故障不会影响整个系统\n5. 可扩展性：可以根据需要独立扩展特定服务",
	}

	if response, exists := responses[msg]; exists {
		return response
	}

	return fmt.Sprintf("您提到了：%s\n\n这是一个很有趣的话题。作为AI助手，我会根据我的知识库为您提供相关信息。不过，我的回答可能不够完整，如果您需要更详细的信息，建议您查阅更多专业资料。\n\n有什么具体的问题我可以帮您解答吗？", msg)
}

// 将文本分割为流式输出
func (l *WebsocketLogic) splitTextForStreaming(text string) []string {
	runes := []rune(text)
	var parts []string
	chunkSize := 10 // 每次输出10个字符

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		parts = append(parts, string(runes[i:end]))
	}

	return parts
}

// 序列化事件
func (l *WebsocketLogic) marshalEvent(event types.StreamEvent) string {
	data, err := json.Marshal(event)
	if err != nil {
		l.Errorf("Marshal event failed: %v", err)
		return "{}"
	}
	return string(data)
}
