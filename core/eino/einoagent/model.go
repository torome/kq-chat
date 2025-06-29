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
	"context"
	"fmt"
	"github.com/cloudwego/eino/components/model"
	"os"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
)

func newChatModel(ctx context.Context) (cm model.BaseChatModel, err error) {
	// TODO Modify component configuration here.
	config := &openai.ChatModelConfig{
		BaseURL: os.Getenv("ARK_BASEURL"),
		Model:   os.Getenv("ARK_CHAT_MODEL"),
		APIKey:  os.Getenv("ARK_API_KEY"),
		//Region:  "cn-beijing",
	}
	fmt.Printf("========> config: %+v\n", config)
	cm, err = openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func newChatModel2(ctx context.Context) (cm *deepseek.ChatModel, err error) {
	// TODO Modify component configuration here.
	cm, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:  "sk-6a62ca496378452b8a9df93402c688e6",
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-chat",
	})
	return cm, nil
}
