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
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/google/uuid"
)

type Action string

const (
	ActionAdd    Action = "add"
	ActionGet    Action = "get"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionList   Action = "list"
)

type Task struct {
	ID        string `json:"id" jsonschema:"description=id of the task"`
	Title     string `json:"title" jsonschema:"description=title of the task"`
	Content   string `json:"content" jsonschema:"description=content of the task"`
	Completed bool   `json:"completed" jsonschema:"description=completed status of the task"`
	Deadline  string `json:"deadline" jsonschema:"description=deadline of the task"`
	IsDeleted bool   `json:"is_deleted" jsonschema:"-"`

	CreatedAt string `json:"created_at" jsonschema:"description=created time of the task"`
}

type TaskRequest struct {
	Action Action      `json:"action" jsonschema:"description=action to perform, enum:add,update,delete,list"`
	Task   *Task       `json:"task" jsonschema:"description=task to add, update, or delete"`
	List   *ListParams `json:"list" jsonschema:"description=list parameters"`
}

type ListParams struct {
	Query  string `json:"query" jsonschema:"description=query to search"`
	IsDone *bool  `json:"is_done" jsonschema:"description=filter by completed status"`
	Limit  *int   `json:"limit" jsonschema:"description=limit the number of results"`
}

type TaskResponse struct {
	Status string `json:"status" jsonschema:"description=status of the response"`

	TaskList []*Task `json:"task_list" jsonschema:"description=list of tasks"`

	Error string `json:"error" jsonschema:"description=error message"`

	Summary string `json:"summary" jsonschema:"description=operation summary for AI context"`
}

type TaskToolImpl struct {
	config *TaskToolConfig
}

type TaskToolConfig struct {
	Storage *Storage
}

func defaultTaskToolConfig(ctx context.Context) (*TaskToolConfig, error) {
	config := &TaskToolConfig{
		Storage: GetDefaultStorage(),
	}
	return config, nil
}

func NewTaskToolImpl(ctx context.Context, config *TaskToolConfig) (*TaskToolImpl, error) {
	var err error
	if config == nil {
		config, err = defaultTaskToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	if config.Storage == nil {
		return nil, fmt.Errorf("storage cannot be empty")
	}

	t := &TaskToolImpl{config: config}

	return t, nil
}

func NewTaskTool(ctx context.Context, config *TaskToolConfig) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultTaskToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	if config.Storage == nil {
		return nil, fmt.Errorf("storage cannot be empty")
	}

	t := &TaskToolImpl{config: config}
	tn, err = t.ToEinoTool()
	if err != nil {
		return nil, err
	}
	return tn, nil
}

func (t *TaskToolImpl) ToEinoTool() (tool.BaseTool, error) {
	description := `
任务管理工具，支持完整的任务生命周期管理。

功能说明：
- add: 创建新任务，需要提供title（必需）和content
- update: 更新现有任务，需要提供id（必需）和要更新的字段
- delete: 删除任务，需要提供id（必需）
- list: 列出任务，可选择性过滤

重要提示：
1. 删除或更新任务时，如果用户没有提供具体的任务ID，请先查看对话历史中最近的任务操作结果
2. 在任务操作结果中可以找到任务ID、标题等关键信息
3. 当用户说"刚才的任务"、"上一个任务"时，使用历史中最近创建的任务ID
4. 操作成功后会返回详细的任务信息，包含完整的任务ID

使用示例：
- 创建：{"action": "add", "task": {"title": "吃西瓜", "content": "晚上吃西瓜"}}
- 删除：{"action": "delete", "task": {"id": "从历史记录中获取的任务ID"}}
- 更新：{"action": "update", "task": {"id": "任务ID", "completed": true}}
`
	return utils.InferTool("task_manager", description, t.Invoke)
}

// 同时优化Invoke方法，在响应中提供更清晰的信息
func (t *TaskToolImpl) Invoke(ctx context.Context, req *TaskRequest) (res *TaskResponse, err error) {
	res = &TaskResponse{}

	switch req.Action {
	case ActionAdd:
		if req.Task == nil {
			res.Status = "error"
			res.Error = "task is required for add action"
			return res, nil
		}
		if req.Task.Title == "" {
			res.Status = "error"
			res.Error = "title is required"
			return res, nil
		}
		req.Task.ID = uuid.New().String()
		if err := t.config.Storage.Add(req.Task); err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("failed to add task: %v", err)
			return res, nil
		}
		res.TaskList = []*Task{req.Task}
		// 添加操作摘要，便于AI理解
		res.Summary = fmt.Sprintf("成功创建任务：%s（ID: %s）", req.Task.Title, req.Task.ID)

	case ActionUpdate:
		if req.Task == nil {
			res.Status = "error"
			res.Error = "task is required for update action"
			return res, nil
		}
		if req.Task.ID == "" {
			res.Status = "error"
			res.Error = "id is required for update action"
			return res, nil
		}
		if err := t.config.Storage.Update(req.Task); err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("failed to update task: %v", err)
			return res, nil
		}
		// 获取更新后的任务信息
		tasks, _ := t.config.Storage.List(&ListParams{})
		for _, task := range tasks {
			if task.ID == req.Task.ID {
				res.TaskList = []*Task{task}
				break
			}
		}
		res.Summary = fmt.Sprintf("成功更新任务：%s（ID: %s）", req.Task.Title, req.Task.ID)

	case ActionDelete:
		if req.Task == nil || req.Task.ID == "" {
			res.Status = "error"
			res.Error = "task id is required for delete action"
			return res, nil
		}
		if err := t.config.Storage.Delete(req.Task.ID); err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("failed to delete task: %v", err)
			return res, nil
		}
		res.Summary = fmt.Sprintf("成功删除任务（ID: %s）", req.Task.ID)

	case ActionList:
		if req.List == nil {
			req.List = &ListParams{}
		}
		tasks, err := t.config.Storage.List(req.List)
		if err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("failed to list tasks: %v", err)
			return res, nil
		}
		res.TaskList = tasks
		res.Summary = fmt.Sprintf("成功查询到 %d 个任务", len(tasks))

	default:
		res.Status = "error"
		res.Error = fmt.Sprintf("unknown action: %s", req.Action)
		return res, nil
	}

	res.Status = "success"
	return res, nil
}
