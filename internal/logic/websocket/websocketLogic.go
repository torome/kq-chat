package websocket

import (
	"ai-agent/internal/svc/agent"
	"ai-agent/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"net/http"

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

// 处理发送消息 - 接入真实的 Agent 能力并处理工具事件
func (l *WebsocketLogic) handleSendMessage(data interface{}) {
	dataBytes, _ := json.Marshal(data)
	var sendData types.SendMessageReq
	json.Unmarshal(dataBytes, &sendData)

	l.Infof("Handling sendMessage: %+v", sendData)

	// 生成记录ID
	recordId := fmt.Sprintf("record_%d", time.Now().UnixNano())

	// 使用真实的 Agent 处理
	l.handleAgentStreamResponseWithTools(sendData, recordId)
}

// 使用 Agent 进行真实的流式响应并处理工具事件
func (l *WebsocketLogic) handleAgentStreamResponseWithTools(data types.SendMessageReq, recordId string) {
	// 1. 如果启用搜索，发送搜索事件（可选）
	if data.SearchEnable {
		searchEvent := types.StreamEvent{
			Type:     "search",
			RecordId: recordId,
			Role:     "assistant",
			SearchInfo: &types.SearchInfo{
				SearchResults: []types.SearchResult{
					{Title: "正在搜索相关资料...", Url: ""},
				},
			},
		}
		l.sendEvent(searchEvent)
	}

	// 2. 发送思考过程事件
	if l.svcCtx.Config.ThinkingEnabled {
		thinkingEvent := types.StreamEvent{
			Type:             "thinking",
			ReasoningContent: "正在分析您的问题并搜索相关信息...",
			RecordId:         recordId,
			Role:             "assistant",
		}
		l.sendEvent(thinkingEvent)
	}

	// 3. 调用 Agent 进行流式处理（带工具事件）
	conversationID := data.ConversationId
	if conversationID == "" {
		conversationID = recordId
	}

	// 调用 Agent 客户端（带工具事件）
	streamWrapper, err := l.svcCtx.AgentClient.StreamWithToolEvents(
		l.ctx,
		conversationID,
		data.Msg,
	)
	if err != nil {
		l.Errorf("Error calling agent: %v", err)
		// 发送错误事件
		errorEvent := types.StreamEvent{
			Type:     "text",
			Content:  fmt.Sprintf("抱歉，处理您的请求时出现错误：%v", err),
			RecordId: recordId,
			Role:     "assistant",
		}
		l.sendEvent(errorEvent)
		l.sendFinishEvent(recordId)
		return
	}
	defer streamWrapper.Close()

	// 4. 同时处理 Agent 的流式输出和工具事件
	var fullContent strings.Builder
	streamFinished := false

	// 创建协程处理工具事件
	go l.handleToolEvents(streamWrapper.ToolEvents())

	for !streamFinished {
		select {
		case <-l.ctx.Done():
			l.Info("Context cancelled, stopping stream")
			return
		default:
			msg, err := streamWrapper.Recv()
			if err == io.EOF {
				l.Infof("Agent stream completed for record: %s", recordId)
				streamFinished = true
				break
			}
			if err != nil {
				l.Errorf("Error receiving from agent (record: %s): %v", recordId, err)
				// 发送错误并结束
				errorEvent := types.StreamEvent{
					Type:     "text",
					Content:  "抱歉，处理过程中出现了问题。",
					RecordId: recordId,
					Role:     "assistant",
				}
				l.sendEvent(errorEvent)
				streamFinished = true
				break
			}

			// 检查消息内容是否为空
			if msg.Content == "" {
				l.Debugf("Received empty content message, skipping (record: %s)", recordId)
				continue // 跳过空内容的消息
			}

			// 累积完整内容
			fullContent.WriteString(msg.Content)

			// 发送文本事件
			textEvent := types.StreamEvent{
				Type:     "text",
				Content:  msg.Content,
				RecordId: recordId,
				Role:     "assistant",
			}
			l.sendEvent(textEvent)
		}
	}

	// 5. 发送结束事件
	l.sendFinishEvent(recordId)
	l.Infof("Message handling completed for record: %s, content length: %d", recordId, fullContent.Len())
}

// handleToolEvents 处理工具事件
func (l *WebsocketLogic) handleToolEvents(toolEvents <-chan agent.ToolEvent) {
	for toolEvent := range toolEvents {
		l.Infof("Received tool event: %s", toolEvent.Type)

		switch toolEvent.Type {
		case "tool-call":
			if event, ok := toolEvent.Event.(types.StreamEvent); ok {
				// 工具调用开始，callResult为false
				event.CallResult = false
				l.sendEvent(event)
				l.Infof("Sent tool-call event for record: %s", toolEvent.RecordId)
			}
		case "tool-result":
			if event, ok := toolEvent.Event.(types.StreamEvent); ok {
				// 工具调用完成，设置callResult为true
				event.CallResult = true
				l.sendEvent(event)
				l.Infof("Sent tool-result event with callResult=true for record: %s", toolEvent.RecordId)
			}
		default:
			l.Errorf("Unknown tool event type: %s", toolEvent.Type)
		}
	}
	l.Info("Tool events channel closed")
}

// 发送单个事件
func (l *WebsocketLogic) sendEvent(event types.StreamEvent) {
	eventData := fmt.Sprintf("data: %s\n\n", l.marshalEvent(event))
	if err := l.conn.WriteMessage(websocket.TextMessage, []byte(eventData)); err != nil {
		l.Errorf("Write message failed: %v", err)
	}
}

// 发送结束事件
func (l *WebsocketLogic) sendFinishEvent(recordId string) {
	finishEvent := types.StreamEvent{
		Type:         "finish",
		RecordId:     recordId,
		Role:         "assistant",
		FinishReason: "stop",
	}
	l.sendEvent(finishEvent)

	// 发送结束标志
	endData := "data: [DONE]\n\n"
	l.conn.WriteMessage(websocket.TextMessage, []byte(endData))
}

// 处理推荐问题（保持原有逻辑）
func (l *WebsocketLogic) handleRecommendQuestions(data interface{}) {
	dataBytes, _ := json.Marshal(data)
	var recommendData types.GetRecommendQuestionsReq
	json.Unmarshal(dataBytes, &recommendData)

	l.Infof("Handling getRecommendQuestions: %+v", recommendData)

	// 可以用 Agent 生成推荐问题，或保持原有逻辑
	questions := l.generateRecommendQuestions(recommendData)

	for _, question := range questions {
		time.Sleep(100 * time.Millisecond)
		if err := l.conn.WriteMessage(websocket.TextMessage, []byte(question+"\n")); err != nil {
			l.Errorf("Write recommend question failed: %v", err)
			return
		}
	}
}

// 生成推荐问题
func (l *WebsocketLogic) generateRecommendQuestions(data types.GetRecommendQuestionsReq) []string {
	// 可以基于 Agent 的能力生成更智能的推荐问题
	// 或者保持现有的静态问题
	return []string{
		"您还想了解什么其他信息？",
		"需要我为您详细解释某个概念吗？",
		"有什么具体的问题需要深入探讨？",
	}
}

// 序列化事件（保持原有逻辑）
func (l *WebsocketLogic) marshalEvent(event types.StreamEvent) string {
	data, err := json.Marshal(event)
	if err != nil {
		l.Errorf("Marshal event failed: %v", err)
		return "{}"
	}
	return string(data)
}
