package elasticsearch

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/indices/create"
	"github.com/elastic/go-elasticsearch/v8/typedapi/indices/exists"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

var (
	client   *elasticsearch.Client
	initOnce sync.Once
)

func Init() error {
	var err error
	initOnce.Do(func() {
		err = initElasticsearchClient()
	})
	return err
}

func initElasticsearchClient() error {
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

	var err error
	client, err = elasticsearch.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// 测试连接
	ctx := context.Background()
	res, err := client.Ping(client.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch ping failed with status: %s", res.Status())
	}

	return nil
}

func GetClient() *elasticsearch.Client {
	return client
}

// CreateIndexIfNotExists 创建索引（如果不存在）
func CreateIndexIfNotExists(ctx context.Context, indexName string) error {
	// 检查索引是否存在
	existsResp, err := exists.NewExistsFunc(client)(indexName).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}

	if existsResp {
		return nil // 索引已存在
	}

	// 创建索引
	_, err = create.NewCreateFunc(client)(indexName).Request(&create.Request{
		Mappings: &types.TypeMapping{
			Properties: map[string]types.Property{
				ContentField:       types.NewTextProperty(),
				MetadataField:      types.NewTextProperty(),
				KnowledgeNameField: types.NewKeywordProperty(),
				ExtraField:         types.NewTextProperty(),
				ContentVectorField: &types.DenseVectorProperty{
					Dims:  of(4096), // 根据embedding模型维度调整
					Index: of(true),
					//Similarity: of("cosine"),
				},
				QAVectorField: &types.DenseVectorProperty{
					Dims:  of(4096),
					Index: of(true),
					//Similarity: of("cosine"),
				},
			},
		},
	}).Do(ctx)

	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

func of[T any](value T) *T {
	return &value
}
