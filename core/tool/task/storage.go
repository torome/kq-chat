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

package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var defaultStorage *Storage

type Storage struct {
	conn sqlx.SqlConn
}

// TaskModel - 对应数据库表结构
type TaskModel struct {
	Id        string         `db:"id"`
	Title     string         `db:"title"`
	Content   sql.NullString `db:"content"`
	Completed bool           `db:"completed"`
	Deadline  sql.NullString `db:"deadline"`
	IsDeleted bool           `db:"is_deleted"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

// 转换方法：数据库模型 -> 业务模型
func (tm *TaskModel) ToTask() *Task {
	task := &Task{
		ID:        tm.Id,
		Title:     tm.Title,
		Completed: tm.Completed,
		IsDeleted: tm.IsDeleted,
		CreatedAt: tm.CreatedAt.Format(time.RFC3339),
	}

	if tm.Content.Valid {
		task.Content = tm.Content.String
	}

	if tm.Deadline.Valid {
		task.Deadline = tm.Deadline.String
	}

	return task
}

// 转换方法：业务模型 -> 数据库模型
func (t *Task) ToTaskModel() *TaskModel {
	tm := &TaskModel{
		Id:        t.ID,
		Title:     t.Title,
		Completed: t.Completed,
		IsDeleted: t.IsDeleted,
	}

	if t.Content != "" {
		tm.Content = sql.NullString{String: t.Content, Valid: true}
	}

	if t.Deadline != "" {
		tm.Deadline = sql.NullString{String: t.Deadline, Valid: true}
	}

	return tm
}

func GetDefaultStorage() *Storage {
	if defaultStorage == nil {
		panic("storage not initialized, call InitDefaultStorage first")
	}
	return defaultStorage
}

// 修改：接受数据库连接字符串而不是目录路径
func InitDefaultStorage(dataSource string) error {
	s, err := NewStorage(dataSource)
	if err != nil {
		return err
	}
	defaultStorage = s
	return nil
}

func NewStorage(dataSource string) (*Storage, error) {
	sqlConn := sqlx.NewMysql(dataSource)
	s := &Storage{
		conn: sqlConn,
	}

	// 测试连接

	// 初始化数据库表
	if err := s.initTables(); err != nil {
		return nil, fmt.Errorf("failed to init tables: %v", err)
	}

	return s, nil
}

// 初始化数据库表
func (s *Storage) initTables() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS tasks (
		id VARCHAR(36) PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		content TEXT,
		completed BOOLEAN DEFAULT FALSE,
		deadline VARCHAR(50),
		is_deleted BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		KEY idx_created_at (created_at),
		KEY idx_completed (completed),
		KEY idx_is_deleted (is_deleted)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	_, err := s.conn.ExecCtx(context.Background(), createTableSQL)
	return err
}

func (s *Storage) Add(task *Task) error {
	// 使用 squirrel 构建 SQL
	query, args, err := squirrel.Insert("tasks").
		Columns("id", "user_id", "title", "content", "completed", "deadline", "is_deleted", "created_at", "updated_at").
		Values(task.ID, task.UserId, task.Title, task.Content, task.Completed, task.Deadline, false, time.Now(), time.Now()).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build insert query: %v", err)
	}

	_, err = s.conn.ExecCtx(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert task: %v", err)
	}

	// 设置创建时间
	task.CreatedAt = time.Now().Format(time.RFC3339)
	task.IsDeleted = false

	return nil
}

func (s *Storage) Update(task *Task) error {
	// 构建动态更新查询
	updateBuilder := squirrel.Update("tasks").
		Where(squirrel.Eq{"id": task.ID, "is_deleted": false, "user_id": task.UserId}).
		Set("updated_at", time.Now()).
		PlaceholderFormat(squirrel.Question)

	// 只更新非空字段
	if task.Title != "" {
		updateBuilder = updateBuilder.Set("title", task.Title)
	}
	if task.Content != "" {
		updateBuilder = updateBuilder.Set("content", task.Content)
	}
	if task.Deadline != "" {
		updateBuilder = updateBuilder.Set("deadline", task.Deadline)
	}
	// completed 字段总是更新（包括 false 值）
	updateBuilder = updateBuilder.Set("completed", task.Completed)

	query, args, err := updateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %v", err)
	}

	result, err := s.conn.ExecCtx(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("failed to update task: %v", err)
	}

	// 检查是否有行被更新
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	return nil
}

func (s *Storage) Delete(task *Task) error {
	// 软删除
	query, args, err := squirrel.Update("tasks").
		Set("is_deleted", true).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": task.ID, "is_deleted": false, "user_id": task.UserId}).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build delete query: %v", err)
	}

	result, err := s.conn.ExecCtx(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	return nil
}

func (s *Storage) List(params *ListParams) ([]*Task, error) {
	// 使用 squirrel 构建复杂查询
	queryBuilder := squirrel.Select("id", "title", "content", "completed", "deadline", "is_deleted", "created_at", "updated_at").
		From("tasks").
		Where(squirrel.Eq{"is_deleted": false}).
		PlaceholderFormat(squirrel.Question)

	// 添加搜索条件
	if params.Query != "" {
		searchTerm := "%" + params.Query + "%"
		queryBuilder = queryBuilder.Where(
			squirrel.Or{
				squirrel.Like{"title": searchTerm},
				squirrel.Like{"content": searchTerm},
			},
		)
	}

	// 添加完成状态过滤
	if params.IsDone != nil {
		queryBuilder = queryBuilder.Where(squirrel.Eq{"completed": *params.IsDone})
	}

	if params.UserId != nil {
		queryBuilder = queryBuilder.Where(squirrel.Eq{"user_id": *params.UserId})
	}

	// 排序：未完成的在前，按创建时间倒序
	queryBuilder = queryBuilder.OrderBy("completed ASC", "created_at DESC")

	// 添加限制
	if params.Limit != nil && *params.Limit > 0 {
		queryBuilder = queryBuilder.Limit(uint64(*params.Limit))
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %v", err)
	}

	var taskModels []TaskModel
	err = s.conn.QueryRowsCtx(context.Background(), &taskModels, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %v", err)
	}

	// 转换为业务模型
	tasks := make([]*Task, 0, len(taskModels))
	for _, tm := range taskModels {
		tasks = append(tasks, tm.ToTask())
	}

	return tasks, nil
}

// GetByID 根据 ID 获取单个任务
func (s *Storage) GetByID(id string) (*Task, error) {
	query, args, err := squirrel.Select("id", "title", "content", "completed", "deadline", "is_deleted", "created_at", "updated_at").
		From("tasks").
		Where(squirrel.Eq{"id": id, "is_deleted": false}).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %v", err)
	}

	var taskModel TaskModel
	err = s.conn.QueryRowCtx(context.Background(), &taskModel, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to query task: %v", err)
	}

	return taskModel.ToTask(), nil
}

// Count 获取任务总数
func (s *Storage) Count(params *ListParams) (int64, error) {
	queryBuilder := squirrel.Select("COUNT(*)").
		From("tasks").
		Where(squirrel.Eq{"is_deleted": false}).
		PlaceholderFormat(squirrel.Question)

	// 添加搜索条件
	if params.Query != "" {
		searchTerm := "%" + params.Query + "%"
		queryBuilder = queryBuilder.Where(
			squirrel.Or{
				squirrel.Like{"title": searchTerm},
				squirrel.Like{"content": searchTerm},
			},
		)
	}

	// 添加完成状态过滤
	if params.IsDone != nil {
		queryBuilder = queryBuilder.Where(squirrel.Eq{"completed": *params.IsDone})
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build count query: %v", err)
	}

	var count int64
	err = s.conn.QueryRowCtx(context.Background(), &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count tasks: %v", err)
	}

	return count, nil
}
