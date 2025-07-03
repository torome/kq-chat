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

package einoagent

import (
	"ai-agent/core/tool/einotool"
	"ai-agent/core/tool/gitclone"
	"ai-agent/core/tool/mcp"
	"ai-agent/core/tool/open"
	"ai-agent/core/tool/task"
	"ai-agent/core/tool/vocabulary"
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/components/tool"
)

var mcpManager *mcp.Manager

func init() {
	// 初始化 MCP 管理器，默认连接到本地 MCP 服务器
	mcpManager = mcp.NewManager("http://localhost:8080/sse")
}

func GetTools(ctx context.Context) ([]tool.BaseTool, error) {
	var allTools []tool.BaseTool

	// 初始化基础工具
	if einoAssistantTool, err := NewEinoAssistantTool(ctx); err == nil {
		allTools = append(allTools, einoAssistantTool)
	} else {
		fmt.Printf("Warning: Failed to load EinoAssistantTool: %v", err)
	}

	if toolTask, err := NewTaskTool(ctx); err == nil {
		allTools = append(allTools, toolTask)
	} else {
		fmt.Printf("Warning: Failed to load TaskTool: %v", err)
	}

	// 添加词汇工具
	if vocabularyTool, err := NewVocabularyTool(ctx); err == nil {
		allTools = append(allTools, vocabularyTool)
	} else {
		fmt.Printf("Warning: Failed to load VocabularyTool: %v", err)
	}

	if toolOpen, err := NewOpenFileTool(ctx); err == nil {
		allTools = append(allTools, toolOpen)
	} else {
		fmt.Printf("Warning: Failed to load OpenFileTool: %v", err)
	}

	if toolGitClone, err := NewGitCloneFile(ctx); err == nil {
		allTools = append(allTools, toolGitClone)
	} else {
		fmt.Printf("Warning: Failed to load GitCloneTool: %v", err)
	}

	if toolDDGSearch, err := NewDDGSearch(ctx, nil); err == nil {
		allTools = append(allTools, toolDDGSearch)
	} else {
		fmt.Printf("Warning: Failed to load DDGSearchTool: %v", err)
	}

	// 尝试获取 MCP 工具
	mcpTools, err := GetMCPTools(ctx)
	if err != nil {
		fmt.Printf("Failed to get MCP tools: %v", err)
	} else {
		allTools = append(allTools, mcpTools...)
		fmt.Printf("Successfully loaded %d MCP tools", len(mcpTools))
	}

	fmt.Printf("Total tools loaded: %d", len(allTools))
	return allTools, nil
}

func defaultDDGSearchConfig(ctx context.Context) (*duckduckgo.Config, error) {
	config := &duckduckgo.Config{}
	return config, nil
}

func NewDDGSearch(ctx context.Context, config *duckduckgo.Config) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultDDGSearchConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	tn, err = duckduckgo.NewTool(ctx, config)
	if err != nil {
		return nil, err
	}
	return tn, nil
}

func NewOpenFileTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return open.NewOpenFileTool(ctx, nil)
}

func NewGitCloneFile(ctx context.Context) (tn tool.BaseTool, err error) {
	return gitclone.NewGitCloneFile(ctx, nil)
}

func NewEinoAssistantTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return einotool.NewEinoAssistantTool(ctx, nil)
}

func NewTaskTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return task.NewTaskTool(ctx, nil)
}

// GetMCPTools 获取 MCP 工具
func GetMCPTools(ctx context.Context) ([]tool.BaseTool, error) {
	if mcpManager == nil {
		return nil, fmt.Errorf("MCP manager not initialized")
	}

	// 检查 MCP 服务器是否健康
	if !mcpManager.IsHealthy(ctx) {
		fmt.Printf("MCP server is not healthy, trying to reconnect...")
		// 可以在这里添加重连逻辑
	}

	return mcpManager.GetTools(ctx)
}

func NewVocabularyTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return vocabulary.NewVocabularyTool(ctx, nil)
}
