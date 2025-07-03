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
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type Storage struct {
	conn sqlx.SqlConn
}

func NewStorage(dataSource string) (*Storage, error) {
	sqlConn := sqlx.NewMysql(dataSource)
	s := &Storage{
		conn: sqlConn,
	}

	if err := s.initTables(); err != nil {
		return nil, fmt.Errorf("failed to init tables: %v", err)
	}

	return s, nil
}

func (s *Storage) initTables() error {
	// 创建单词表
	wordsTable := `
	CREATE TABLE IF NOT EXISTS words (
		id INT AUTO_INCREMENT PRIMARY KEY,
		word VARCHAR(100) NOT NULL UNIQUE,
		phonetic VARCHAR(200),
		meaning TEXT NOT NULL,
		example TEXT,
		difficulty TINYINT DEFAULT 1,
		category VARCHAR(50),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		KEY idx_word (word),
		KEY idx_difficulty (difficulty),
		KEY idx_category (category)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	// 创建用户进度表
	progressTable := `
	CREATE TABLE IF NOT EXISTS user_word_progress (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(100) NOT NULL,
		word_id INT NOT NULL,
		status ENUM('new', 'learning', 'reviewing', 'mastered') DEFAULT 'new',
		correct_count INT DEFAULT 0,
		wrong_count INT DEFAULT 0,
		last_review_at DATETIME,
		next_review_at DATETIME,
		review_interval INT DEFAULT 1,
		ease_factor DECIMAL(3,2) DEFAULT 2.50,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		UNIQUE KEY uk_user_word (user_id, word_id),
		KEY idx_user_status (user_id, status),
		KEY idx_next_review (user_id, next_review_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	// 创建学习会话表
	sessionsTable := `
	CREATE TABLE IF NOT EXISTS learning_sessions (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(100) NOT NULL,
		session_type ENUM('review', 'learn') NOT NULL,
		total_words INT DEFAULT 0,
		correct_answers INT DEFAULT 0,
		wrong_answers INT DEFAULT 0,
		duration_seconds INT DEFAULT 0,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		KEY idx_user_session (user_id, started_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	tables := []string{wordsTable, progressTable, sessionsTable}

	for _, table := range tables {
		_, err := s.conn.ExecCtx(context.Background(), table)
		if err != nil {
			return err
		}
	}

	return nil
}

// 添加单词到词库
func (s *Storage) AddWord(word *Word) error {
	query, args, err := squirrel.Insert("words").
		Columns("word", "phonetic", "meaning", "example", "difficulty", "category").
		Values(word.Word, word.Phonetic, word.Meaning, word.Example, word.Difficulty, word.Category).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build insert query: %v", err)
	}

	_, err = s.conn.ExecCtx(context.Background(), query, args...)
	return err
}

// 获取待复习单词数量
func (s *Storage) GetPendingReviewCount(userID int64) (int, error) {
	query, args, err := squirrel.Select("COUNT(*)").
		From("user_word_progress").
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.Eq{"status": "reviewing"}).
		Where(squirrel.LtOrEq{"next_review_at": time.Now()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	if err != nil {
		return 0, err
	}

	var count int
	err = s.conn.QueryRowCtx(context.Background(), &count, query, args...)
	return count, err
}

// 获取下一个要学习的单词
func (s *Storage) GetNextWordToStudy(userID int64) (*CurrentWord, error) {
	// 首先查找需要复习的单词
	reviewWord, err := s.getNextReviewWord(userID)
	if err != nil {
		return nil, err
	}
	if reviewWord != nil {
		return reviewWord, nil
	}

	// 如果没有复习的，获取新单词
	return s.getNextNewWord(userID)
}

// 获取下一个复习单词
func (s *Storage) getNextReviewWord(userID int64) (*CurrentWord, error) {
	query := `
	SELECT w.id, w.word, w.phonetic, w.meaning, w.example, w.difficulty, w.category,
	       p.id as progress_id, p.status, p.correct_count, p.wrong_count, 
	       p.last_review_at, p.next_review_at, p.review_interval, p.ease_factor
	FROM words w
	JOIN user_word_progress p ON w.id = p.word_id
	WHERE p.user_id = ? AND p.status = 'reviewing' AND p.next_review_at <= ?
	ORDER BY p.next_review_at ASC
	LIMIT 1`

	var wordModel struct {
		// Word fields
		ID         int    `db:"id"`
		Word       string `db:"word"`
		Phonetic   string `db:"phonetic"`
		Meaning    string `db:"meaning"`
		Example    string `db:"example"`
		Difficulty int    `db:"difficulty"`
		Category   string `db:"category"`
		// Progress fields
		ProgressID     string       `db:"progress_id"`
		Status         string       `db:"status"`
		CorrectCount   int          `db:"correct_count"`
		WrongCount     int          `db:"wrong_count"`
		LastReviewAt   sql.NullTime `db:"last_review_at"`
		NextReviewAt   sql.NullTime `db:"next_review_at"`
		ReviewInterval int          `db:"review_interval"`
		EaseFactor     float64      `db:"ease_factor"`
	}

	err := s.conn.QueryRowCtx(context.Background(), &wordModel, query, userID, time.Now())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 没有需要复习的单词
		}
		return nil, err
	}

	word := &Word{
		ID:         wordModel.ID,
		Word:       wordModel.Word,
		Phonetic:   wordModel.Phonetic,
		Meaning:    wordModel.Meaning,
		Example:    wordModel.Example,
		Difficulty: wordModel.Difficulty,
		Category:   wordModel.Category,
	}

	progress := &UserWordProgress{
		ID:             wordModel.ProgressID,
		UserID:         userID,
		WordID:         wordModel.ID,
		Status:         wordModel.Status,
		CorrectCount:   wordModel.CorrectCount,
		WrongCount:     wordModel.WrongCount,
		ReviewInterval: wordModel.ReviewInterval,
		EaseFactor:     wordModel.EaseFactor,
	}

	if wordModel.LastReviewAt.Valid {
		progress.LastReviewAt = wordModel.LastReviewAt.Time
	}
	if wordModel.NextReviewAt.Valid {
		progress.NextReviewAt = wordModel.NextReviewAt.Time
	}

	return &CurrentWord{
		Word:        word,
		Progress:    progress,
		SessionType: "review",
		IsReview:    true,
		ShowMeaning: false, // 复习模式不显示中文
	}, nil
}

// 获取下一个新单词
func (s *Storage) getNextNewWord(userID int64) (*CurrentWord, error) {
	// 查找用户还没有学习过的单词
	query := `
	SELECT w.id, w.word, w.phonetic, w.meaning, w.example, w.difficulty, w.category
	FROM words w
	LEFT JOIN user_word_progress p ON w.id = p.word_id AND p.user_id = ?
	WHERE p.id IS NULL
	ORDER BY w.difficulty ASC, w.id ASC
	LIMIT 1`

	var wordModel struct {
		ID         int    `db:"id"`
		Word       string `db:"word"`
		Phonetic   string `db:"phonetic"`
		Meaning    string `db:"meaning"`
		Example    string `db:"example"`
		Difficulty int    `db:"difficulty"`
		Category   string `db:"category"`
	}

	err := s.conn.QueryRowCtx(context.Background(), &wordModel, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 没有新单词了
		}
		return nil, err
	}

	word := &Word{
		ID:         wordModel.ID,
		Word:       wordModel.Word,
		Phonetic:   wordModel.Phonetic,
		Meaning:    wordModel.Meaning,
		Example:    wordModel.Example,
		Difficulty: wordModel.Difficulty,
		Category:   wordModel.Category,
	}

	// 为新单词创建学习进度记录
	progressID := uuid.New().String()
	_, err = s.conn.ExecCtx(context.Background(),
		"INSERT INTO user_word_progress (id, user_id, word_id, status) VALUES (?, ?, ?, 'learning')",
		progressID, userID, wordModel.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create progress: %v", err)
	}

	return &CurrentWord{
		Word:        word,
		SessionType: "learn",
		IsReview:    false,
		ShowMeaning: true, // 学习模式显示中文
	}, nil
}

// 提交答案（核心遗忘曲线算法）
func (s *Storage) SubmitAnswer(userID int64, wordID int, userAnswer string) (*AnswerSubmission, error) {
	// 获取正确答案
	word, err := s.getWordByID(wordID)
	if err != nil {
		return nil, err
	}

	// 判断答案是否正确（支持多个正确答案，用分号分隔）
	correctAnswers := strings.Split(word.Meaning, ";")
	isCorrect := false
	for _, correct := range correctAnswers {
		if strings.TrimSpace(strings.ToLower(userAnswer)) == strings.TrimSpace(strings.ToLower(correct)) {
			isCorrect = true
			break
		}
	}

	// 更新学习进度
	err = s.updateProgress(userID, wordID, isCorrect)
	if err != nil {
		return nil, err
	}

	return &AnswerSubmission{
		UserID:        userID,
		WordID:        wordID,
		UserAnswer:    userAnswer,
		IsCorrect:     isCorrect,
		CorrectAnswer: word.Meaning,
	}, nil
}

// 更新学习进度（遗忘曲线核心算法）
func (s *Storage) updateProgress(userID int64, wordID int, isCorrect bool) error {
	// 获取当前进度
	progress, err := s.getUserWordProgress(userID, wordID)
	if err != nil {
		return err
	}

	now := time.Now()

	// 更新正确/错误次数
	if isCorrect {
		progress.CorrectCount++
	} else {
		progress.WrongCount++
	}

	// 计算新的遗忘曲线参数
	progress.LastReviewAt = now
	progress = s.calculateNextReview(progress, isCorrect)

	// 更新数据库
	query, args, err := squirrel.Update("user_word_progress").
		Set("correct_count", progress.CorrectCount).
		Set("wrong_count", progress.WrongCount).
		Set("last_review_at", progress.LastReviewAt).
		Set("next_review_at", progress.NextReviewAt).
		Set("review_interval", progress.ReviewInterval).
		Set("ease_factor", progress.EaseFactor).
		Set("status", progress.Status).
		Where(squirrel.Eq{"user_id": userID, "word_id": wordID}).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	if err != nil {
		return err
	}

	_, err = s.conn.ExecCtx(context.Background(), query, args...)
	return err
}

// 遗忘曲线算法（基于SM-2算法改进）
func (s *Storage) calculateNextReview(progress *UserWordProgress, isCorrect bool) *UserWordProgress {
	if isCorrect {
		// 答对了
		if progress.CorrectCount == 1 {
			// 第一次答对，1天后复习
			progress.ReviewInterval = 1
		} else if progress.CorrectCount == 2 {
			// 第二次答对，6天后复习
			progress.ReviewInterval = 6
		} else {
			// 之后根据遗忘曲线计算
			progress.ReviewInterval = int(math.Ceil(float64(progress.ReviewInterval) * progress.EaseFactor))
		}

		// 调整难度系数
		progress.EaseFactor = progress.EaseFactor + (0.1 - (5.0-4.0)*0.08 - (5.0-4.0)*(5.0-4.0)*0.02)
		if progress.EaseFactor < 1.3 {
			progress.EaseFactor = 1.3
		}

		// 更新状态
		if progress.CorrectCount >= 3 && progress.EaseFactor >= 2.5 {
			progress.Status = "mastered"
		} else {
			progress.Status = "reviewing"
		}
	} else {
		// 答错了，重置到1天后复习
		progress.ReviewInterval = 1
		progress.EaseFactor = progress.EaseFactor - 0.2
		if progress.EaseFactor < 1.3 {
			progress.EaseFactor = 1.3
		}
		progress.Status = "reviewing"
	}

	// 计算下次复习时间
	progress.NextReviewAt = progress.LastReviewAt.AddDate(0, 0, progress.ReviewInterval)

	return progress
}

// 获取用户单词进度
func (s *Storage) getUserWordProgress(userID int64, wordID int) (*UserWordProgress, error) {
	query, args, err := squirrel.Select("id", "user_id", "word_id", "status", "correct_count", "wrong_count",
		"last_review_at", "next_review_at", "review_interval", "ease_factor").
		From("user_word_progress").
		Where(squirrel.Eq{"user_id": userID, "word_id": wordID}).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	if err != nil {
		return nil, err
	}

	var progressModel struct {
		ID             string       `db:"id"`
		UserID         int64        `db:"user_id"`
		WordID         int          `db:"word_id"`
		Status         string       `db:"status"`
		CorrectCount   int          `db:"correct_count"`
		WrongCount     int          `db:"wrong_count"`
		LastReviewAt   sql.NullTime `db:"last_review_at"`
		NextReviewAt   sql.NullTime `db:"next_review_at"`
		ReviewInterval int          `db:"review_interval"`
		EaseFactor     float64      `db:"ease_factor"`
	}

	err = s.conn.QueryRowCtx(context.Background(), &progressModel, query, args...)
	if err != nil {
		return nil, err
	}

	progress := &UserWordProgress{
		ID:             progressModel.ID,
		UserID:         progressModel.UserID,
		WordID:         progressModel.WordID,
		Status:         progressModel.Status,
		CorrectCount:   progressModel.CorrectCount,
		WrongCount:     progressModel.WrongCount,
		ReviewInterval: progressModel.ReviewInterval,
		EaseFactor:     progressModel.EaseFactor,
	}

	if progressModel.LastReviewAt.Valid {
		progress.LastReviewAt = progressModel.LastReviewAt.Time
	}
	if progressModel.NextReviewAt.Valid {
		progress.NextReviewAt = progressModel.NextReviewAt.Time
	}

	return progress, nil
}

// 根据ID获取单词
func (s *Storage) getWordByID(wordID int) (*Word, error) {
	query, args, err := squirrel.Select("id", "word", "phonetic", "meaning", "example", "difficulty", "category").
		From("words").
		Where(squirrel.Eq{"id": wordID}).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	if err != nil {
		return nil, err
	}

	var wordModel struct {
		ID         int    `db:"id"`
		Word       string `db:"word"`
		Phonetic   string `db:"phonetic"`
		Meaning    string `db:"meaning"`
		Example    string `db:"example"`
		Difficulty int    `db:"difficulty"`
		Category   string `db:"category"`
	}

	err = s.conn.QueryRowCtx(context.Background(), &wordModel, query, args...)
	if err != nil {
		return nil, err
	}

	word := &Word{
		ID:         wordModel.ID,
		Word:       wordModel.Word,
		Phonetic:   wordModel.Phonetic,
		Meaning:    wordModel.Meaning,
		Example:    wordModel.Example,
		Difficulty: wordModel.Difficulty,
		Category:   wordModel.Category,
	}

	return word, nil
}

// 创建学习会话
func (s *Storage) CreateSession(session *LearningSession) error {
	query, args, err := squirrel.Insert("learning_sessions").
		Columns("id", "user_id", "session_type", "started_at").
		Values(session.ID, session.UserID, session.SessionType, session.StartedAt).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	if err != nil {
		return err
	}

	_, err = s.conn.ExecCtx(context.Background(), query, args...)
	return err
}

// 获取学习统计
func (s *Storage) GetLearningStats(userID int64) (*LearningStats, error) {
	stats := &LearningStats{UserID: userID}

	// 获取词库总数
	err := s.conn.QueryRowCtx(context.Background(), &stats.TotalWords, "SELECT COUNT(*) FROM words")
	if err != nil {
		return nil, err
	}

	// 获取已学习单词数
	err = s.conn.QueryRowCtx(context.Background(), &stats.LearnedWords,
		"SELECT COUNT(*) FROM user_word_progress WHERE user_id = ? AND status != 'new'", userID)
	if err != nil {
		return nil, err
	}

	// 获取已掌握单词数
	err = s.conn.QueryRowCtx(context.Background(), &stats.MasteredWords,
		"SELECT COUNT(*) FROM user_word_progress WHERE user_id = ? AND status = 'mastered'", userID)
	if err != nil {
		return nil, err
	}

	// 获取待复习数量
	err = s.conn.QueryRowCtx(context.Background(), &stats.PendingReviewCount,
		"SELECT COUNT(*) FROM user_word_progress WHERE user_id = ? AND status = 'reviewing' AND next_review_at <= ?",
		userID, time.Now())
	if err != nil {
		return nil, err
	}

	// 获取今日学习统计
	today := time.Now().Format("2006-01-02")
	err = s.conn.QueryRowCtx(context.Background(), &stats.TodayReviewCount,
		"SELECT COALESCE(SUM(correct_answers + wrong_answers), 0) FROM learning_sessions WHERE user_id = ? AND session_type = 'review' AND DATE(started_at) = ?",
		userID, today)
	if err != nil {
		return nil, err
	}

	err = s.conn.QueryRowCtx(context.Background(), &stats.TodayNewCount,
		"SELECT COALESCE(SUM(correct_answers + wrong_answers), 0) FROM learning_sessions WHERE user_id = ? AND session_type = 'learn' AND DATE(started_at) = ?",
		userID, today)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
