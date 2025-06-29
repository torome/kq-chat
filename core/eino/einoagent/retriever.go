package einoagent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/retriever/es8"
	"github.com/cloudwego/eino-ext/components/retriever/es8/search_mode"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"os"

	espkg "ai-agent/core/elasticsearch"
)

// newRetriever component initialization function of node 'ElasticsearchRetriever' in graph 'EinoAgent'
func newRetriever(ctx context.Context) (rtr retriever.Retriever, err error) {
	indexName := os.Getenv("ELASTICSEARCH_INDEX_NAME")
	if indexName == "" {
		indexName = "eino-knowledge-base"
	}

	config := &es8.RetrieverConfig{
		Client: espkg.GetClient(),
		Index:  indexName,
		SearchMode: search_mode.SearchModeDenseVectorSimilarity(
			search_mode.DenseVectorSimilarityTypeCosineSimilarity,
			espkg.ContentVectorField,
		),
		ResultParser: func(ctx context.Context, hit types.Hit) (*schema.Document, error) {
			doc := &schema.Document{
				ID:       *hit.Id_,
				MetaData: map[string]any{},
			}

			var src map[string]any
			if err := json.Unmarshal(hit.Source_, &src); err != nil {
				return nil, fmt.Errorf("failed to unmarshal source: %w", err)
			}

			// 处理各个字段
			for field, val := range src {
				switch field {
				case espkg.ContentField:
					if content, ok := val.(string); ok {
						doc.Content = content
					}
				case espkg.MetadataField:
					if metadataStr, ok := val.(string); ok {
						var metadata map[string]any
						if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
							for k, v := range metadata {
								doc.MetaData[k] = v
							}
						}
					}
				case espkg.ContentVectorField:
					if vector, ok := val.([]interface{}); ok {
						var v []float64
						for _, item := range vector {
							if f, ok := item.(float64); ok {
								v = append(v, f)
							}
						}
						doc.WithDenseVector(v)
					}
				case espkg.KnowledgeNameField:
					if knowledgeName, ok := val.(string); ok {
						doc.MetaData[espkg.KnowledgeNameField] = knowledgeName
					}
				}
			}

			// 设置相似度分数
			if hit.Score_ != nil {
				doc.WithScore(float64(*hit.Score_))
			}

			return doc, nil
		},
	}

	embeddingIns, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	config.Embedding = embeddingIns

	rtr, err = es8.NewRetriever(ctx, config)
	if err != nil {
		return nil, err
	}
	return rtr, nil
}
