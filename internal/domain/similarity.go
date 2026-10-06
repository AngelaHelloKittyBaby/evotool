package domain

// SimilarityAction 表示相似度检测后的建议动作。
type SimilarityAction string

const (
	// SimilarityActionNone 表示无需动作。
	SimilarityActionNone SimilarityAction = "none"
	// SimilarityActionRecordDuplicate 表示仅记录重复代码。
	SimilarityActionRecordDuplicate SimilarityAction = "record_duplicate"
	// SimilarityActionProposeLibrary 表示建议抽取共享 Library。
	SimilarityActionProposeLibrary SimilarityAction = "propose_library"
	// SimilarityActionRefactor 表示建议进入重构流程。
	SimilarityActionRefactor SimilarityAction = "refactor"
)

// SimilarityReport 表示代码或能力相似度检测结果。
type SimilarityReport struct {
	SourceA         string
	SourceB         string
	SimilarityScore float64
	Reason          string
	SuggestedAction SimilarityAction
}
