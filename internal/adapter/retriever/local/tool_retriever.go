package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

const defaultRoot = ".evotool"

type ToolRetriever struct {
	root string
}

func NewToolRetriever(root string) *ToolRetriever {
	root = strings.TrimSpace(root)
	if root == "" {
		root = defaultRoot
	}
	return &ToolRetriever{root: root}
}

func (r *ToolRetriever) Retrieve(ctx context.Context, query domain.RetrievalQuery) (domain.RetrievalResult, error) {
	if err := checkContext(ctx); err != nil {
		return domain.RetrievalResult{}, err
	}

	tools, err := r.readTools(ctx)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	semanticCandidates := r.semanticRetrieve(query, tools)
	semanticCandidates = limitCandidates(semanticCandidates, positiveOrDefault(query.SemanticTopK, defaultSemanticTopK))

	filtered := filterCandidates(query, semanticCandidates)
	filterLimit := positiveOrDefault(query.FilterTopK, defaultFilterTopK)
	if query.Limit > 0 && query.Limit < filterLimit {
		filterLimit = query.Limit
	}
	filtered = limitCandidates(filtered, filterLimit)

	return domain.RetrievalResult{
		Query:      query,
		Candidates: filtered,
		StageReports: []domain.RetrievalStageReport{
			{
				Stage:       domain.RetrievalStageSemanticRetrieval,
				InputCount:  len(tools),
				OutputCount: len(semanticCandidates),
				Reason:      "local keyword score over manifest metadata",
			},
			{
				Stage:       domain.RetrievalStageMetadataFilter,
				InputCount:  len(semanticCandidates),
				OutputCount: len(filtered),
				Reason:      "filtered by category, runtime, io schema, lifecycle, trust, and permissions",
			},
		},
	}, nil
}

func (r *ToolRetriever) readTools(ctx context.Context) ([]domain.Tool, error) {
	dir := filepath.Join(r.root, "registry", "metadata", "tools")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read tool metadata directory: %w", err)
	}

	tools := make([]domain.Tool, 0, len(entries))
	for _, entry := range entries {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read tool metadata %s: %w", entry.Name(), err)
		}
		var tool domain.Tool
		if err := json.Unmarshal(data, &tool); err != nil {
			return nil, fmt.Errorf("decode tool metadata %s: %w", entry.Name(), err)
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

func (r *ToolRetriever) semanticRetrieve(query domain.RetrievalQuery, tools []domain.Tool) []domain.ToolCandidate {
	queryTokens := tokenSet(queryText(query))
	candidates := make([]domain.ToolCandidate, 0, len(tools))
	for _, tool := range tools {
		score, matches := scoreTool(queryTokens, tool)
		if score <= 0 && len(queryTokens) > 0 {
			continue
		}
		candidate := domain.ToolCandidate{
			Tool:       tool,
			Score:      score,
			UsageStats: tool.UsageStats,
			RankScore: domain.CapabilityRankScore{
				SemanticSimilarity: score,
				EnvironmentMatch:   environmentMatch(query, tool),
			},
			MatchReason:      matchReason(matches),
			Source:           domain.RetrievalSourceKeyword,
			DependencyHealth: dependencyHealth(tool),
		}
		candidates = append(candidates, candidate)
	}
	sortCandidates(candidates)
	return candidates
}

func scoreTool(queryTokens map[string]struct{}, tool domain.Tool) (float64, []string) {
	if len(queryTokens) == 0 {
		return 1, nil
	}

	documentTokens := tokenSet(toolText(tool))
	matches := make([]string, 0, len(queryTokens))
	for token := range queryTokens {
		if _, ok := documentTokens[token]; ok {
			matches = append(matches, token)
		}
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return 0, nil
	}

	coverage := float64(len(matches)) / float64(len(queryTokens))
	nameBonus := nameMatchBonus(queryTokens, tool)
	score := coverage + nameBonus
	if score > 1 {
		score = 1
	}
	return score, matches
}

func nameMatchBonus(queryTokens map[string]struct{}, tool domain.Tool) float64 {
	nameTokens := tokenSet(strings.Join([]string{tool.ID, tool.Name, tool.Manifest.Name}, " "))
	if len(nameTokens) == 0 {
		return 0
	}
	matches := 0
	for token := range queryTokens {
		if _, ok := nameTokens[token]; ok {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	return 0.15 * float64(matches) / float64(len(queryTokens))
}

func filterCandidates(query domain.RetrievalQuery, candidates []domain.ToolCandidate) []domain.ToolCandidate {
	filtered := make([]domain.ToolCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if metadataMatches(query, candidate) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func metadataMatches(query domain.RetrievalQuery, candidate domain.ToolCandidate) bool {
	tool := candidate.Tool
	filters := query.Filters
	if filters.MinScore > 0 && candidate.Score < filters.MinScore {
		return false
	}
	if len(query.Categories) > 0 && !matchesAnyText(toolCategories(tool), query.Categories) {
		return false
	}
	if strings.TrimSpace(query.Domain) != "" && !matchesAnyText(toolDomainText(tool), []string{query.Domain}) {
		return false
	}
	if len(query.InputTypes) > 0 && !matchesAllTerms(toolInputText(tool), query.InputTypes) {
		return false
	}
	if len(query.OutputTypes) > 0 && !matchesAllTerms(toolOutputText(tool), query.OutputTypes) {
		return false
	}
	if runtime := strings.TrimSpace(firstNonEmpty(filters.Runtime, query.Environment.Runtime)); runtime != "" && !sameText(runtime, toolRuntime(tool)) {
		return false
	}
	if filters.NetworkAllowed != nil && !*filters.NetworkAllowed && toolUsesNetwork(tool) {
		return false
	}
	if len(filters.AllowedStatuses) > 0 && !containsLifecycle(filters.AllowedStatuses, tool.LifecycleStatus) {
		return false
	}
	if len(filters.AllowedTrustLevels) > 0 && !containsTrust(filters.AllowedTrustLevels, tool.TrustLevel) {
		return false
	}
	if filters.RequireHealthyDeps && candidate.DependencyHealth != domain.DependencyHealthy {
		return false
	}
	if filters.RequirePolicyAllowed && (tool.LifecycleStatus == domain.LifecycleQuarantined || tool.LifecycleStatus == domain.LifecycleDeleted) {
		return false
	}
	return true
}

func queryText(query domain.RetrievalQuery) string {
	parts := []string{query.Intent, query.Domain, query.Environment.Runtime}
	parts = append(parts, query.Categories...)
	parts = append(parts, query.InputTypes...)
	parts = append(parts, query.OutputTypes...)
	parts = append(parts, query.Examples...)
	for key, value := range query.Metadata {
		parts = append(parts, key, value)
	}
	return strings.Join(parts, " ")
}

func toolText(tool domain.Tool) string {
	parts := []string{
		tool.ID,
		tool.Name,
		tool.Description,
		tool.Category,
		tool.Manifest.Name,
		tool.Manifest.Description,
		tool.Manifest.Category,
		tool.Manifest.Runtime,
		tool.Manifest.EntryPoint,
		tool.Manifest.Environment.Runtime,
		tool.Manifest.Environment.OS,
		tool.Manifest.Environment.Arch,
	}
	parts = append(parts, tool.Manifest.Tags...)
	parts = append(parts, tool.Manifest.Capabilities...)
	for _, parameter := range tool.Manifest.Inputs {
		parts = append(parts, parameter.Name, parameter.Type, parameter.Description)
	}
	for _, parameter := range tool.Manifest.Outputs {
		parts = append(parts, parameter.Name, parameter.Type, parameter.Description)
	}
	for _, example := range tool.Manifest.Examples {
		parts = append(parts, example.Input, example.Output, example.Description)
	}
	for _, dependency := range append(tool.Dependencies, tool.Manifest.Dependencies...) {
		parts = append(parts, dependency.ID, dependency.Name, string(dependency.Kind), dependency.Version)
	}
	for _, dependency := range tool.Manifest.LibraryRefs {
		parts = append(parts, dependency.ID, dependency.Name, string(dependency.Kind), dependency.Version)
	}
	parts = append(parts, schemaText(tool.Manifest.InputSchema), schemaText(tool.Manifest.OutputSchema))
	return strings.Join(parts, " ")
}

func toolInputText(tool domain.Tool) string {
	parts := make([]string, 0, len(tool.Manifest.Inputs)*3+1)
	for _, parameter := range tool.Manifest.Inputs {
		parts = append(parts, parameter.Name, parameter.Type, parameter.Description)
	}
	parts = append(parts, schemaText(tool.Manifest.InputSchema))
	return strings.Join(parts, " ")
}

func toolOutputText(tool domain.Tool) string {
	parts := make([]string, 0, len(tool.Manifest.Outputs)*3+1)
	for _, parameter := range tool.Manifest.Outputs {
		parts = append(parts, parameter.Name, parameter.Type, parameter.Description)
	}
	parts = append(parts, schemaText(tool.Manifest.OutputSchema))
	return strings.Join(parts, " ")
}

func toolCategories(tool domain.Tool) string {
	return strings.Join([]string{tool.Category, tool.Manifest.Category}, " ")
}

func toolDomainText(tool domain.Tool) string {
	parts := []string{tool.Category, tool.Manifest.Category, tool.Description, tool.Manifest.Description}
	parts = append(parts, tool.Manifest.Tags...)
	parts = append(parts, tool.Manifest.Capabilities...)
	return strings.Join(parts, " ")
}

func schemaText(schema map[string]any) string {
	if len(schema) == 0 {
		return ""
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return ""
	}
	return string(data)
}

func tokenSet(value string) map[string]struct{} {
	tokens := make(map[string]struct{})
	for _, token := range tokenize(value) {
		tokens[token] = struct{}{}
	}
	return tokens
}

func tokenize(value string) []string {
	value = strings.ToLower(value)
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			tokens = append(tokens, field)
		}
	}
	return tokens
}

func matchesAnyText(text string, terms []string) bool {
	for _, term := range terms {
		if matchesTerm(text, term) {
			return true
		}
	}
	return false
}

func matchesAllTerms(text string, terms []string) bool {
	for _, term := range terms {
		if !matchesTerm(text, term) {
			return false
		}
	}
	return true
}

func matchesTerm(text string, term string) bool {
	term = strings.TrimSpace(term)
	if term == "" {
		return true
	}
	if strings.Contains(strings.ToLower(text), strings.ToLower(term)) {
		return true
	}
	textTokens := tokenSet(text)
	for _, token := range tokenize(term) {
		if _, ok := textTokens[token]; !ok {
			return false
		}
	}
	return true
}

func sameText(a string, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func toolRuntime(tool domain.Tool) string {
	return firstNonEmpty(tool.Manifest.Runtime, tool.Manifest.Environment.Runtime)
}

func toolUsesNetwork(tool domain.Tool) bool {
	if tool.Manifest.Environment.Network {
		return true
	}
	return tool.Manifest.Permissions.Network == domain.NetworkPolicyRestricted || tool.Manifest.Permissions.Network == domain.NetworkPolicyFull
}

func environmentMatch(query domain.RetrievalQuery, tool domain.Tool) float64 {
	runtime := firstNonEmpty(query.Filters.Runtime, query.Environment.Runtime)
	if runtime == "" {
		return 1
	}
	if sameText(runtime, toolRuntime(tool)) {
		return 1
	}
	return 0
}

func dependencyHealth(tool domain.Tool) domain.DependencyHealthStatus {
	if len(tool.Dependencies) == 0 && len(tool.Manifest.Dependencies) == 0 && len(tool.Manifest.LibraryRefs) == 0 {
		return domain.DependencyHealthy
	}
	return domain.DependencyUnknown
}

func containsLifecycle(allowed []domain.LifecycleStatus, status domain.LifecycleStatus) bool {
	for _, item := range allowed {
		if item == status {
			return true
		}
	}
	return false
}

func containsTrust(allowed []domain.TrustLevel, trust domain.TrustLevel) bool {
	for _, item := range allowed {
		if item == trust {
			return true
		}
	}
	return false
}

func matchReason(matches []string) string {
	if len(matches) == 0 {
		return "no query terms provided"
	}
	if len(matches) > 6 {
		matches = matches[:6]
	}
	return "matched terms: " + strings.Join(matches, ", ")
}

func sortCandidates(candidates []domain.ToolCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Tool.Name < candidates[j].Tool.Name
		}
		return candidates[i].Score > candidates[j].Score
	})
}

func limitCandidates(candidates []domain.ToolCandidate, limit int) []domain.ToolCandidate {
	if limit <= 0 || len(candidates) <= limit {
		return candidates
	}
	return candidates[:limit]
}

func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
