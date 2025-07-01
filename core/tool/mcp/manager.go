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

package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	mcpext "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type Manager struct {
	client    *client.Client
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	once      sync.Once
	initErr   error
	serverURL string
}

func NewManager(serverURL string) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		serverURL: serverURL,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (m *Manager) Initialize() error {
	m.once.Do(func() {
		log.Printf("Initializing MCP client for server: %s", m.serverURL)

		// 创建 MCP 客户端
		cli, err := client.NewSSEMCPClient(m.serverURL)
		if err != nil {
			m.initErr = fmt.Errorf("failed to create MCP client: %w", err)
			return
		}

		// 启动客户端
		err = cli.Start(m.ctx)
		if err != nil {
			m.initErr = fmt.Errorf("failed to start MCP client: %w", err)
			return
		}

		// 初始化连接
		initCtx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		defer cancel()

		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "kq-chat-client",
			Version: "1.0.0",
		}

		_, err = cli.Initialize(initCtx, initRequest)
		if err != nil {
			cli.Close()
			m.initErr = fmt.Errorf("failed to initialize MCP client: %w", err)
			return
		}

		log.Printf("MCP client initialized successfully")

		m.mu.Lock()
		m.client = cli
		m.mu.Unlock()
	})

	return m.initErr
}

func (m *Manager) GetTools(ctx context.Context) ([]tool.BaseTool, error) {
	// 确保客户端已初始化
	if err := m.Initialize(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	cli := m.client
	m.mu.RUnlock()

	if cli == nil {
		return nil, fmt.Errorf("MCP client is not initialized")
	}

	// 使用带超时的上下文
	toolCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 获取工具列表
	tools, err := mcpext.GetTools(toolCtx, &mcpext.Config{Cli: cli})
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP tools: %w", err)
	}

	log.Printf("Successfully loaded %d MCP tools", len(tools))
	return tools, nil
}

func (m *Manager) IsHealthy(ctx context.Context) bool {
	m.mu.RLock()
	cli := m.client
	m.mu.RUnlock()

	if cli == nil {
		return false
	}

	// 检查上下文是否已取消
	select {
	case <-m.ctx.Done():
		return false
	default:
	}

	// 尝试 ping
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := cli.Ping(pingCtx)
	return err == nil
}

func (m *Manager) Close() error {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		err := m.client.Close()
		m.client = nil
		return err
	}

	return nil
}
