// core/eino/einoagent/retriever.go
package einoagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/retriever/es8"
	"github.com/cloudwego/eino-ext/components/retriever/es8/search_mode"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"

	espkg "ai-agent/core/elasticsearch"
)

// newRetriever component initialization function of node 'ElasticsearchRetriever' in graph 'EinoAgent'
func newRetriever(ctx context.Context) (rtr retriever.Retriever, err error) {
	log.Printf("Initializing Elasticsearch retriever...")

	// 尝试使用全局客户端
	err = espkg.Init()
	if err != nil {
		log.Printf("Global elasticsearch init failed: %v, creating new client", err)
	}

	client := espkg.GetClient()
	if client == nil {
		log.Printf("Global client is nil, creating new elasticsearch client")

		// 创建新的客户端
		esAddr := os.Getenv("ELASTICSEARCH_ADDR")
		if esAddr == "" {
			esAddr = "http://localhost:9200"
		}

		esUsername := os.Getenv("ELASTICSEARCH_USERNAME")
		esPassword := os.Getenv("ELASTICSEARCH_PASSWORD")

		config := elasticsearch.Config{
			Addresses: []string{esAddr},
		}

		if esUsername != "" && esPassword != "" {
			config.Username = esUsername
			config.Password = esPassword
		}

		client, err = elasticsearch.NewClient(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
		}

		log.Printf("Created new Elasticsearch client successfully")
	} else {
		log.Printf("Using global Elasticsearch client")
	}

	indexName := os.Getenv("ELASTICSEARCH_INDEX_NAME")
	if indexName == "" {
		indexName = "eino-knowledge-base"
	}

	log.Printf("Using Elasticsearch index: %s", indexName)

	config := &es8.RetrieverConfig{
		Client: client,
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
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}
	config.Embedding = embeddingIns

	rtr, err = es8.NewRetriever(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ES retriever: %w", err)
	}

	log.Printf("Elasticsearch retriever initialized successfully")
	return rtr, nil
}
