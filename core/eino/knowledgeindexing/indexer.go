package knowledgeindexing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/indexer/es8"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	espkg "ai-agent/core/elasticsearch"
)

func init() {
	err := espkg.Init()
	if err != nil {
		log.Fatalf("failed to init elasticsearch: %v", err)
	}
}

// newIndexer component initialization function of node 'ElasticsearchIndexer' in graph 'KnowledgeIndexing'
func newIndexer(ctx context.Context) (idr indexer.Indexer, err error) {
	//esAddr := os.Getenv("ELASTICSEARCH_ADDR")
	//esUsername := os.Getenv("ELASTICSEARCH_USERNAME")
	//esPassword := os.Getenv("ELASTICSEARCH_PASSWORD")
	indexName := os.Getenv("ELASTICSEARCH_INDEX_NAME")
	if indexName == "" {
		indexName = "eino-knowledge-base"
	}

	// 确保索引存在
	err = espkg.CreateIndexIfNotExists(ctx, indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	config := &es8.IndexerConfig{
		Client:    espkg.GetClient(),
		Index:     indexName,
		BatchSize: 10,
		DocumentToFields: func(ctx context.Context, doc *schema.Document) (map[string]es8.FieldValue, error) {
			if doc.ID == "" {
				doc.ID = uuid.New().String()
			}

			metadataBytes, err := json.Marshal(doc.MetaData)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}

			fields := map[string]es8.FieldValue{
				espkg.ContentField: {
					Value:    doc.Content,
					EmbedKey: espkg.ContentVectorField,
				},
				espkg.MetadataField: {
					Value: string(metadataBytes),
				},
			}

			// 添加知识库名称（如果存在）
			if knowledgeName, ok := ctx.Value("knowledge_name").(string); ok {
				fields[espkg.KnowledgeNameField] = es8.FieldValue{Value: knowledgeName}
			}

			return fields, nil
		},
	}

	embeddingIns, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	config.Embedding = embeddingIns

	idr, err = es8.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}
	return idr, nil
}
