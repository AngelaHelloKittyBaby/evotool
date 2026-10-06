package domain

// RetrievalQuery 表示一次能力检索请求。
type RetrievalQuery struct {
	Intent      string
	Categories  []string
	InputTypes  []string
	OutputTypes []string
	Domain      string
	Limit       int
	Filters     RetrievalFilters
	Metadata    map[string]string
}

// RetrievalFilters 表示能力检索中的过滤条件。
type RetrievalFilters struct {
	MinScore             float64
	AllowedStatuses      []LifecycleStatus
	AllowedTrustLevels   []TrustLevel
	MaxRiskLevel         RiskLevel
	RequireHealthyDeps   bool
	RequirePolicyAllowed bool
}

// RetrievalSource 表示候选工具的召回来源。
type RetrievalSource string

const (
	// RetrievalSourceKeyword 表示通过关键词召回。
	RetrievalSourceKeyword RetrievalSource = "keyword"
	// RetrievalSourceEmbedding 表示通过向量语义召回。
	RetrievalSourceEmbedding RetrievalSource = "embedding"
	// RetrievalSourceCategory 表示通过分类召回。
	RetrievalSourceCategory RetrievalSource = "category"
	// RetrievalSourceHistory 表示通过执行历史召回。
	RetrievalSourceHistory RetrievalSource = "history"
)
