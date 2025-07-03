package elasticsearch

import (
	"context"
	"fmt"
	"log"
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
	initErr  error
)

func Init() error {
	initOnce.Do(func() {
		initErr = initElasticsearchClient()
	})
	return initErr
}

func initElasticsearchClient() error {
	esAddr := os.Getenv("ELASTICSEARCH_ADDR")
	fmt.Printf("esAddr:%s", esAddr)
	if esAddr == "" {
		esAddr = "http://127.0.0.1:9200"
	}

	esUsername := os.Getenv("ELASTICSEARCH_USERNAME")
	esPassword := os.Getenv("ELASTICSEARCH_PASSWORD")
	fmt.Printf("esUsername:%s", esUsername)
	config := elasticsearch.Config{
		Addresses: []string{esAddr},
	}

	if esUsername != "" && esPassword != "" {
		config.Username = "elastic"
		config.Password = "7U+IOR_ESxuMfJ2D=45P"
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
		return fmt.Errorf("elasticsearch ping failed with status: %s", res.Status())
	}

	log.Printf("Elasticsearch client initialized successfully, connected to: %s", esAddr)
	return nil
}

func GetClient() *elasticsearch.Client {
	if client == nil {
		log.Printf("Elasticsearch client is nil, attempting to initialize...")
		if err := Init(); err != nil {
			log.Printf("Failed to initialize Elasticsearch client: %v", err)
			return nil
		}
	}
	return client
}

// MustGetClient 获取客户端，如果为空则panic
func MustGetClient() *elasticsearch.Client {
	client := GetClient()
	if client == nil {
		log.Fatal("Elasticsearch client is not initialized")
	}
	return client
}

// CreateIndexIfNotExists 创建索引（如果不存在）
func CreateIndexIfNotExists(ctx context.Context, indexName string) error {
	client := MustGetClient()

	// 检查索引是否存在
	existsResp, err := exists.NewExistsFunc(client)(indexName).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}

	if existsResp {
		log.Printf("Index %s already exists", indexName)
		return nil // 索引已存在
	}

	log.Printf("Creating index: %s", indexName)

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

	log.Printf("Index %s created successfully", indexName)
	return nil
}

func of[T any](value T) *T {
	return &value
}
