package domain

import "time"

// RetrievalQuery 表示一次能力检索请求。
type RetrievalQuery struct {
	Intent       string
	Categories   []string
	InputTypes   []string
	OutputTypes  []string
	Domain       string
	Examples     []string
	Environment  EnvironmentSpec
	SemanticTopK int
	FilterTopK   int
	RankTopK     int
	Limit        int
	Filters      RetrievalFilters
	Metadata     map[string]string
}

// RetrievalFilters 表示能力检索中的过滤条件。
type RetrievalFilters struct {
	MinScore             float64
	AllowedStatuses      []LifecycleStatus
	AllowedTrustLevels   []TrustLevel
	MaxRiskLevel         RiskLevel
	RequireHealthyDeps   bool
	RequirePolicyAllowed bool
	Runtime              string
	NetworkAllowed       *bool
}

// RetrievalStage 表示能力检索流水线中的阶段。
type RetrievalStage string

const (
	// RetrievalStageIntentExtraction 表示任务意图提取阶段。
	RetrievalStageIntentExtraction RetrievalStage = "intent_extraction"
	// RetrievalStageSemanticRetrieval 表示语义召回阶段。
	RetrievalStageSemanticRetrieval RetrievalStage = "semantic_retrieval"
	// RetrievalStageMetadataFilter 表示元数据过滤阶段。
	RetrievalStageMetadataFilter RetrievalStage = "metadata_filter"
	// RetrievalStageCapabilityRank 表示能力排序阶段。
	RetrievalStageCapabilityRank RetrievalStage = "capability_rank"
	// RetrievalStageLLMSelection 表示 LLM 或规则最终选择阶段。
	RetrievalStageLLMSelection RetrievalStage = "llm_selection"
)

// RetrievalStageReport 记录一次检索阶段的候选数量变化。
type RetrievalStageReport struct {
	Stage       RetrievalStage
	InputCount  int
	OutputCount int
	Reason      string
}

// RetrievalResult 表示一次完整检索流水线的结果。
type RetrievalResult struct {
	Query        RetrievalQuery
	Candidates   []ToolCandidate
	Selected     *ToolCandidate
	StageReports []RetrievalStageReport
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

// ToolUsageStats 表示工具历史使用经验。
type ToolUsageStats struct {
	UsageCount     int
	SuccessCount   int
	FailureCount   int
	SuccessRate    float64
	AverageLatency time.Duration
	LastUsedAt     time.Time
}

// CapabilityRankScore 表示综合排序分数及其组成部分。
type CapabilityRankScore struct {
	SemanticSimilarity float64
	SuccessRate        float64
	Reliability        float64
	Latency            float64
	UsageHistory       float64
	EnvironmentMatch   float64
	FinalScore         float64
}
