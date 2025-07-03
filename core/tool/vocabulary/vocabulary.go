/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package vocabulary

import (
	"ai-agent/common/ctxdata"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/google/uuid"
)

type Action string

const (
	ActionStartSession Action = "start_session" // 开始学习会话
	ActionGetWord      Action = "get_word"      // 获取当前要学习的单词
	ActionSubmitAnswer Action = "submit_answer" // 提交答案
	ActionGetStats     Action = "get_stats"     // 获取学习统计
	ActionAddWord      Action = "add_word"      // 添加新单词到词库
)

// 单词信息
type Word struct {
	ID         int    `json:"id"`
	Word       string `json:"word"`
	Phonetic   string `json:"phonetic"`
	Meaning    string `json:"meaning"`
	Example    string `json:"example"`
	Difficulty int    `json:"difficulty"`
	Category   string `json:"category"`
}

// 用户学习进度
type UserWordProgress struct {
	ID             string    `json:"id"`
	UserID         int64     `json:"user_id"`
	WordID         int       `json:"word_id"`
	Status         string    `json:"status"` // new, learning, reviewing, mastered
	CorrectCount   int       `json:"correct_count"`
	WrongCount     int       `json:"wrong_count"`
	LastReviewAt   time.Time `json:"last_review_at"`
	NextReviewAt   time.Time `json:"next_review_at"`
	ReviewInterval int       `json:"review_interval"` // 复习间隔（天）
	EaseFactor     float64   `json:"ease_factor"`     // 遗忘曲线系数
}

// 学习会话
type LearningSession struct {
	ID             string    `json:"id"`
	UserID         int64     `json:"user_id"`
	SessionType    string    `json:"session_type"` // review, learn
	TotalWords     int       `json:"total_words"`
	CorrectAnswers int       `json:"correct_answers"`
	WrongAnswers   int       `json:"wrong_answers"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at,omitempty"`
}

// 当前学习的单词（包含学习状态）
type CurrentWord struct {
	Word        *Word             `json:"word"`
	Progress    *UserWordProgress `json:"progress,omitempty"`
	SessionType string            `json:"session_type"` // review 或 learn
	IsReview    bool              `json:"is_review"`
	ShowMeaning bool              `json:"show_meaning"` // 是否显示中文（学习模式为true，复习模式为false）
}

// 答案提交
type AnswerSubmission struct {
	UserID        int64  `json:"user_id"`
	WordID        int    `json:"word_id"`
	UserAnswer    string `json:"user_answer"`
	IsCorrect     bool   `json:"is_correct"`
	CorrectAnswer string `json:"correct_answer"`
}

// 学习统计
type LearningStats struct {
	UserID             int64 `json:"user_id"`
	TotalWords         int   `json:"total_words"`          // 词库总数
	LearnedWords       int   `json:"learned_words"`        // 已学习单词数
	MasteredWords      int   `json:"mastered_words"`       // 已掌握单词数
	TodayReviewCount   int   `json:"today_review_count"`   // 今日复习数量
	TodayNewCount      int   `json:"today_new_count"`      // 今日新学数量
	PendingReviewCount int   `json:"pending_review_count"` // 待复习数量
	StreakDays         int   `json:"streak_days"`          // 连续学习天数
}

type VocabularyRequest struct {
	Action Action `json:"action" jsonschema:"description=action to perform, enum:start_session,get_word,submit_answer,get_stats,add_word"`

	// 通用参数
	UserID int64 `json:"user_id,omitempty" jsonschema:"description=user identifier"`

	// 获取单词相关
	SessionType string `json:"session_type,omitempty" jsonschema:"description=session type: review or learn"`

	// 提交答案相关
	WordID     int    `json:"word_id,omitempty" jsonschema:"description=word id for answer submission"`
	UserAnswer string `json:"user_answer,omitempty" jsonschema:"description=user's answer"`

	// 添加单词相关
	Word *Word `json:"word,omitempty" jsonschema:"description=word to add to vocabulary"`
}

type VocabularyResponse struct {
	Status  string `json:"status" jsonschema:"description=status of the response"`
	Error   string `json:"error,omitempty" jsonschema:"description=error message"`
	Summary string `json:"summary,omitempty" jsonschema:"description=operation summary"`

	// 不同类型的响应数据
	CurrentWord  *CurrentWord      `json:"current_word,omitempty" jsonschema:"description=current word to learn/review"`
	AnswerResult *AnswerSubmission `json:"answer_result,omitempty" jsonschema:"description=answer submission result"`
	Stats        *LearningStats    `json:"stats,omitempty" jsonschema:"description=learning statistics"`
	Session      *LearningSession  `json:"session,omitempty" jsonschema:"description=learning session info"`
}

type VocabularyToolImpl struct {
	config *VocabularyToolConfig
}

type VocabularyToolConfig struct {
	Storage *Storage
}

func defaultVocabularyToolConfig(ctx context.Context) (*VocabularyToolConfig, error) {
	// 从环境变量读取数据库配置，与 task 工具共用
	dataSource := os.Getenv("TASK_DATASOURCE")
	if dataSource == "" {
		dataSource = "root:@tcp(localhost:3306)/p2?charset=utf8mb4&parseTime=true&loc=Local"
		//dataSource = "p2:eT8FTFWjLTFE4Fxx@tcp(127.0.0.1:3306)/p2?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai"
	}

	storage, err := NewStorage(dataSource)
	if err != nil {
		return nil, fmt.Errorf("failed to init vocabulary storage: %v", err)
	}

	config := &VocabularyToolConfig{
		Storage: storage,
	}
	return config, nil
}

func NewVocabularyToolImpl(ctx context.Context, config *VocabularyToolConfig) (*VocabularyToolImpl, error) {
	var err error
	if config == nil {
		config, err = defaultVocabularyToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	if config.Storage == nil {
		return nil, fmt.Errorf("storage cannot be empty")
	}

	v := &VocabularyToolImpl{config: config}
	return v, nil
}

func NewVocabularyTool(ctx context.Context, config *VocabularyToolConfig) (tool.BaseTool, error) {
	var err error
	if config == nil {
		config, err = defaultVocabularyToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	if config.Storage == nil {
		return nil, fmt.Errorf("storage cannot be empty")
	}

	v := &VocabularyToolImpl{config: config}
	return v.ToEinoTool()
}

func (v *VocabularyToolImpl) ToEinoTool() (tool.BaseTool, error) {
	description := `
单词记忆工具，基于遗忘曲线的智能复习系统。

功能说明：
- start_session: 开始学习会话，系统会自动判断是复习还是学习新单词
- get_word: 获取当前要学习的单词（复习优先，然后是新单词）
- submit_answer: 提交答案（复习模式下用户输入中文意思）
- get_stats: 查看学习统计数据
- add_word: 添加新单词到词库

学习流程：
1. 用户开始会话 -> 系统检查是否有待复习单词
2. 如有复习 -> 显示英文+音标，用户输入中文
3. 复习完成 -> 开始学习新单词，显示完整信息
4. 根据答题情况调整复习间隔

使用示例：
- 开始: {"action": "start_session", "user_id": "user123"}
- 获取单词: {"action": "get_word", "user_id": "user123"}
- 提交答案: {"action": "submit_answer", "user_id": "user123", "word_id": 1, "user_answer": "苹果"}
`
	return utils.InferTool("vocabulary_manager", description, v.Invoke)
}

func (v *VocabularyToolImpl) Invoke(ctx context.Context, req *VocabularyRequest) (*VocabularyResponse, error) {
	resp := &VocabularyResponse{}

	userID := ctxdata.GetUidFromCtx(ctx)
	if userID == 0 {
		resp.Status = "error"
		resp.Error = "user_id is required"
		return resp, nil
	}

	req.UserID = userID

	switch req.Action {
	case ActionStartSession:
		return v.handleStartSession(ctx, req)
	case ActionGetWord:
		return v.handleGetWord(ctx, req)
	case ActionSubmitAnswer:
		return v.handleSubmitAnswer(ctx, req)
	case ActionGetStats:
		return v.handleGetStats(ctx, req)
	case ActionAddWord:
		return v.handleAddWord(ctx, req)
	default:
		resp.Status = "error"
		resp.Error = fmt.Sprintf("unknown action: %s", req.Action)
		return resp, nil
	}
}

// 开始学习会话
func (v *VocabularyToolImpl) handleStartSession(ctx context.Context, req *VocabularyRequest) (*VocabularyResponse, error) {
	// 检查是否有待复习的单词
	reviewCount, err := v.config.Storage.GetPendingReviewCount(req.UserID)
	if err != nil {
		return &VocabularyResponse{
			Status: "error",
			Error:  fmt.Sprintf("failed to check pending reviews: %v", err),
		}, nil
	}

	sessionType := "learn"
	if reviewCount > 0 {
		sessionType = "review"
	}

	// 创建学习会话
	session := &LearningSession{
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		SessionType: sessionType,
		StartedAt:   time.Now(),
	}

	err = v.config.Storage.CreateSession(session)
	if err != nil {
		return &VocabularyResponse{
			Status: "error",
			Error:  fmt.Sprintf("failed to create session: %v", err),
		}, nil
	}

	summary := fmt.Sprintf("开始%s会话", map[string]string{"review": "复习", "learn": "学习"}[sessionType])
	if sessionType == "review" {
		summary += fmt.Sprintf("，有 %d 个单词需要复习", reviewCount)
	}

	return &VocabularyResponse{
		Status:  "success",
		Summary: summary,
		Session: session,
	}, nil
}

// 获取当前要学习的单词
func (v *VocabularyToolImpl) handleGetWord(ctx context.Context, req *VocabularyRequest) (*VocabularyResponse, error) {
	// 首先检查是否有待复习的单词
	currentWord, err := v.config.Storage.GetNextWordToStudy(req.UserID)
	if err != nil {
		return &VocabularyResponse{
			Status: "error",
			Error:  fmt.Sprintf("failed to get next word: %v", err),
		}, nil
	}

	if currentWord == nil {
		return &VocabularyResponse{
			Status:  "success",
			Summary: "今日学习任务已完成！",
		}, nil
	}

	return &VocabularyResponse{
		Status:      "success",
		CurrentWord: currentWord,
		Summary:     fmt.Sprintf("当前%s单词：%s", map[bool]string{true: "复习", false: "学习"}[currentWord.IsReview], currentWord.Word.Word),
	}, nil
}

// 提交答案
func (v *VocabularyToolImpl) handleSubmitAnswer(ctx context.Context, req *VocabularyRequest) (*VocabularyResponse, error) {
	if req.WordID == 0 {
		return &VocabularyResponse{
			Status: "error",
			Error:  "word_id is required",
		}, nil
	}

	result, err := v.config.Storage.SubmitAnswer(req.UserID, req.WordID, req.UserAnswer)
	if err != nil {
		return &VocabularyResponse{
			Status: "error",
			Error:  fmt.Sprintf("failed to submit answer: %v", err),
		}, nil
	}

	summary := "回答正确！"
	if !result.IsCorrect {
		summary = fmt.Sprintf("回答错误。正确答案是：%s", result.CorrectAnswer)
	}

	return &VocabularyResponse{
		Status:       "success",
		AnswerResult: result,
		Summary:      summary,
	}, nil
}

// 获取学习统计
func (v *VocabularyToolImpl) handleGetStats(ctx context.Context, req *VocabularyRequest) (*VocabularyResponse, error) {
	stats, err := v.config.Storage.GetLearningStats(req.UserID)
	if err != nil {
		return &VocabularyResponse{
			Status: "error",
			Error:  fmt.Sprintf("failed to get stats: %v", err),
		}, nil
	}

	return &VocabularyResponse{
		Status:  "success",
		Stats:   stats,
		Summary: fmt.Sprintf("已学习 %d/%d 个单词，今日还需复习 %d 个", stats.LearnedWords, stats.TotalWords, stats.PendingReviewCount),
	}, nil
}

// 添加新单词
func (v *VocabularyToolImpl) handleAddWord(ctx context.Context, req *VocabularyRequest) (*VocabularyResponse, error) {
	if req.Word == nil {
		return &VocabularyResponse{
			Status: "error",
			Error:  "word is required",
		}, nil
	}

	err := v.config.Storage.AddWord(req.Word)
	if err != nil {
		return &VocabularyResponse{
			Status: "error",
			Error:  fmt.Sprintf("failed to add word: %v", err),
		}, nil
	}

	return &VocabularyResponse{
		Status:  "success",
		Summary: fmt.Sprintf("成功添加单词：%s", req.Word.Word),
	}, nil
}
