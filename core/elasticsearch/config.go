package elasticsearch

import (
	"github.com/elastic/go-elasticsearch/v8"
)

type Config struct {
	Client         *elasticsearch.Client
	IndexName      string
	APIKey         string
	BaseURL        string
	EmbeddingModel string
	ChatModel      string
}

const (
	// 字段名定义
	ContentField       = "content"
	MetadataField      = "metadata"
	ContentVectorField = "content_vector"
	QAContentField     = "qa_content"
	QAVectorField      = "qa_content_vector"
	KnowledgeNameField = "knowledge_name"
	ExtraField         = "extra"
	DistanceField      = "distance"

	// 索引前缀
	IndexPrefix = "eino:doc:"
)
