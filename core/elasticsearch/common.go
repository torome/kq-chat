package elasticsearch

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

// NewEmbedding 创建embedding实例
func NewEmbedding(ctx context.Context, conf *Config) (embedding.Embedder, error) {
	if conf.APIKey == "" || conf.BaseURL == "" || conf.EmbeddingModel == "" {
		return nil, fmt.Errorf("embedding configuration is incomplete")
	}

	embeddingConfig := &openai.EmbeddingConfig{
		APIKey:  conf.APIKey,
		BaseURL: conf.BaseURL,
		Model:   conf.EmbeddingModel,
	}

	return openai.NewEmbedder(ctx, embeddingConfig)
}
