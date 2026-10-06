package ports

import "context"

// Embedding 表示一段文本或能力描述的向量表示。
type Embedding []float32

// Embedder 为工具、Library 或任务文本生成向量。
type Embedder interface {
	Embed(ctx context.Context, text string) (Embedding, error)
}
