package algo

import (
	"math"
	"time"
)

// SM2Card SM-2算法单词卡片数据结构
type SM2Card struct {
	// 基础信息
	WordId int64 `json:"word_id"`
	UserId int64 `json:"user_id"`

	// SM-2算法核心参数
	EasinessFactor float64 `json:"easiness_factor"` // 难易度因子 (默认2.5)
	Interval       int64   `json:"interval"`        // 复习间隔天数
	Repetitions    int64   `json:"repetitions"`     // 连续正确复习次数

	// 时间相关
	LastReviewTime time.Time `json:"last_review_time"` // 上次复习时间
	NextReviewTime time.Time `json:"next_review_time"` // 下次复习时间

	// 学习状态
	IsLearning  bool `json:"is_learning"`  // 是否在学习阶段
	IsGraduated bool `json:"is_graduated"` // 是否已毕业(间隔>=21天)

	// 统计信息
	TotalReviews   int64 `json:"total_reviews"`   // 总复习次数
	CorrectReviews int64 `json:"correct_reviews"` // 正确复习次数
}

// Quality 回忆质量等级 (0-5)
type Quality int64

const (
	QualityBlackout    Quality = 0 // 完全忘记
	QualityIncorrect   Quality = 1 // 错误,但看到答案后有印象
	QualityIncorrect2  Quality = 2 // 错误,但看到答案后容易想起
	QualityCorrect     Quality = 3 // 正确,但费力
	QualityCorrectEasy Quality = 4 // 正确,不费力
	QualityPerfect     Quality = 5 // 完美回忆
)

// SM2Algorithm SuperMemo SM-2算法实现
type SM2Algorithm struct{}

// NewSM2Algorithm 创建SM-2算法实例
func NewSM2Algorithm() *SM2Algorithm {
	return &SM2Algorithm{}
}

// InitCard 初始化新单词卡片
func (sm2 *SM2Algorithm) InitCard(wordId, userId int64) *SM2Card {
	now := time.Now()
	return &SM2Card{
		WordId:         wordId,
		UserId:         userId,
		EasinessFactor: 2.5, // 初始难易度因子
		Interval:       1,   // 初始间隔1天
		Repetitions:    0,   // 初始复习次数0
		LastReviewTime: now,
		NextReviewTime: now.AddDate(0, 0, 1), // 明天复习
		IsLearning:     true,
		IsGraduated:    false,
		TotalReviews:   0,
		CorrectReviews: 0,
	}
}

// Review 执行复习并更新卡片参数
func (sm2 *SM2Algorithm) Review(card *SM2Card, quality Quality) *SM2Card {
	now := time.Now()

	// 更新统计信息
	card.TotalReviews++
	card.LastReviewTime = now

	// 如果质量>=3，认为回答正确
	if quality >= QualityCorrect {
		card.CorrectReviews++

		if card.Repetitions == 0 {
			// 第一次正确复习
			card.Interval = 1
			card.Repetitions = 1
		} else if card.Repetitions == 1 {
			// 第二次正确复习
			card.Interval = 6
			card.Repetitions = 2
		} else {
			// 第三次及以后正确复习，使用难易度因子计算
			card.Interval = int64(math.Round(float64(card.Interval) * card.EasinessFactor))
			card.Repetitions++
		}

		// 检查是否毕业(间隔>=21天)
		if card.Interval >= 21 {
			card.IsGraduated = true
			card.IsLearning = false
		}

	} else {
		// 回答错误，重置repetitions，但保持已学习的难易度因子
		card.Repetitions = 0
		card.Interval = 1
		card.IsLearning = true
		card.IsGraduated = false
	}

	// 根据质量更新难易度因子
	card.EasinessFactor = sm2.calculateNewEasinessFactor(card.EasinessFactor, quality)

	// 计算下次复习时间
	card.NextReviewTime = sm2.calculateNextReviewTime(now, card.Interval, card.IsLearning)

	return card
}

// calculateNewEasinessFactor 根据回忆质量计算新的难易度因子
func (sm2 *SM2Algorithm) calculateNewEasinessFactor(currentEF float64, quality Quality) float64 {
	// SM-2算法公式: EF' = EF + (0.1 - (5-q) * (0.08 + (5-q) * 0.02))
	q := float64(quality)
	newEF := currentEF + (0.1 - (5-q)*(0.08+(5-q)*0.02))

	// 难易度因子最小值为1.3
	if newEF < 1.3 {
		newEF = 1.3
	}

	return math.Round(newEF*100) / 100 // 保留两位小数
}

// calculateNextReviewTime 计算下次复习时间
func (sm2 *SM2Algorithm) calculateNextReviewTime(lastReview time.Time, interval int64, isLearning bool) time.Time {
	if isLearning && interval == 1 {
		// 学习阶段，间隔更短
		return lastReview.Add(10 * time.Minute) // 10分钟后
	}

	// 正常间隔，按天计算
	return lastReview.AddDate(0, 0, int(interval))
}

// GetDueCards 获取到期需要复习的卡片
func (sm2 *SM2Algorithm) GetDueCards(cards []*SM2Card, currentTime time.Time) []*SM2Card {
	var dueCards []*SM2Card

	for _, card := range cards {
		if currentTime.After(card.NextReviewTime) || currentTime.Equal(card.NextReviewTime) {
			dueCards = append(dueCards, card)
		}
	}

	return dueCards
}

// GetLearningCards 获取学习阶段的卡片
func (sm2 *SM2Algorithm) GetLearningCards(cards []*SM2Card) []*SM2Card {
	var learningCards []*SM2Card

	for _, card := range cards {
		if card.IsLearning {
			learningCards = append(learningCards, card)
		}
	}

	return learningCards
}

// GetGraduatedCards 获取已毕业的卡片
func (sm2 *SM2Algorithm) GetGraduatedCards(cards []*SM2Card) []*SM2Card {
	var graduatedCards []*SM2Card

	for _, card := range cards {
		if card.IsGraduated {
			graduatedCards = append(graduatedCards, card)
		}
	}

	return graduatedCards
}

// GetRetentionRate 计算记忆保持率
func (sm2 *SM2Algorithm) GetRetentionRate(card *SM2Card) float64 {
	if card.TotalReviews == 0 {
		return 0.0
	}

	return float64(card.CorrectReviews) / float64(card.TotalReviews)
}

// ConvertUserResponseToQuality 将用户回答转换为质量等级
func (sm2 *SM2Algorithm) ConvertUserResponseToQuality(isCorrect bool, responseTime int, difficulty string) Quality {
	if !isCorrect {
		return QualityIncorrect // 错误
	}

	// 根据响应时间和难度判断质量
	switch difficulty {
	case "hard":
		return QualityCorrect // 正确但费力
	case "good":
		return QualityCorrectEasy // 正确不费力
	case "easy":
		return QualityPerfect // 完美回忆
	default:
		// 根据响应时间判断
		if responseTime <= 3 {
			return QualityPerfect // 3秒内回答，完美回忆
		} else if responseTime <= 10 {
			return QualityCorrectEasy // 10秒内回答，不费力
		} else {
			return QualityCorrect // 超过10秒，费力
		}
	}
}

// PredictSuccessRate 预测成功率(可选功能)
func (sm2 *SM2Algorithm) PredictSuccessRate(card *SM2Card, daysSinceLastReview int64) float64 {
	// 基于难易度因子和间隔时间预测成功率
	// 这是一个简化的预测模型，实际可以更复杂

	if daysSinceLastReview <= card.Interval {
		// 在预期间隔内，成功率较高
		return 0.9
	} else {
		// 超过预期间隔，成功率下降
		overdueDays := daysSinceLastReview - card.Interval
		successRate := 0.9 * math.Exp(-float64(overdueDays)/float64(card.Interval)*0.5)
		if successRate < 0.1 {
			successRate = 0.1
		}
		return successRate
	}
}

// CalculateWorkload 计算每日学习负担
func (sm2 *SM2Algorithm) CalculateWorkload(cards []*SM2Card, days int) map[string]int {
	workload := make(map[string]int)
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, i).Format("2006-01-02")
		count := 0

		for _, card := range cards {
			reviewDate := card.NextReviewTime.Format("2006-01-02")
			if reviewDate == date {
				count++
			}
		}

		workload[date] = count
	}

	return workload
}
