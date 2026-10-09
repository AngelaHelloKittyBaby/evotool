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

	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

const defaultLibraryLimit = 10

// LibraryRetriever searches reusable libraries from the local registry.
type LibraryRetriever struct {
	root string
}

// NewLibraryRetriever creates a local library retriever rooted at root.
func NewLibraryRetriever(root string) *LibraryRetriever {
	root = strings.TrimSpace(root)
	if root == "" {
		root = defaultRoot
	}
	return &LibraryRetriever{root: root}
}

// SearchLibraries returns libraries that match the query and are compatible with it.
func (r *LibraryRetriever) SearchLibraries(ctx context.Context, query domain.RetrievalQuery) ([]domain.LibraryCandidate, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	libraries, err := r.readLibraries(ctx)
	if err != nil {
		return nil, err
	}

	queryTokens := tokenSet(queryText(query))
	candidates := make([]domain.LibraryCandidate, 0, len(libraries))
	for _, library := range libraries {
		score, matches := scoreLibrary(queryTokens, library)
		if score <= 0 && len(queryTokens) > 0 {
			continue
		}
		if query.Filters.MinScore > 0 && score < query.Filters.MinScore {
			continue
		}
		if !libraryCompatible(query, library) {
			continue
		}
		candidates = append(candidates, domain.LibraryCandidate{
			Library:    library,
			Score:      score,
			Reason:     matchReason(matches),
			Compatible: true,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Library.Name < candidates[j].Library.Name
		}
		return candidates[i].Score > candidates[j].Score
	})
	return limitLibraryCandidates(candidates, libraryLimit(query)), nil
}

func (r *LibraryRetriever) readLibraries(ctx context.Context) ([]domain.Library, error) {
	dir := filepath.Join(r.root, "registry", "metadata", "libraries")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read library metadata directory: %w", err)
	}

	libraries := make([]domain.Library, 0, len(entries))
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
			return nil, fmt.Errorf("read library metadata %s: %w", entry.Name(), err)
		}
		var library domain.Library
		if err := json.Unmarshal(data, &library); err != nil {
			return nil, fmt.Errorf("decode library metadata %s: %w", entry.Name(), err)
		}
		libraries = append(libraries, library)
	}
	return libraries, nil
}

func scoreLibrary(queryTokens map[string]struct{}, library domain.Library) (float64, []string) {
	if len(queryTokens) == 0 {
		return 1, nil
	}

	documentTokens := tokenSet(libraryText(library))
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
	nameBonus := libraryNameMatchBonus(queryTokens, library)
	score := coverage + nameBonus
	if score > 1 {
		score = 1
	}
	return score, matches
}

func libraryNameMatchBonus(queryTokens map[string]struct{}, library domain.Library) float64 {
	nameTokens := tokenSet(strings.Join([]string{library.ID, library.Name, library.Manifest.Name}, " "))
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

func libraryText(library domain.Library) string {
	parts := []string{
		library.ID,
		library.Name,
		library.Description,
		library.Category,
		library.Manifest.Name,
		library.Manifest.Description,
		library.Manifest.Category,
		library.Manifest.Runtime,
	}
	parts = append(parts, library.Manifest.Tags...)
	for _, export := range library.Manifest.Exports {
		parts = append(parts, export.Name, export.Kind, export.Description)
	}
	for _, dependency := range append(library.Dependencies, library.Manifest.Dependencies...) {
		parts = append(parts, dependency.ID, dependency.Name, string(dependency.Kind), dependency.Version)
	}
	return strings.Join(parts, " ")
}

func libraryCompatible(query domain.RetrievalQuery, library domain.Library) bool {
	filters := query.Filters
	if len(query.Categories) > 0 && !matchesAnyText(libraryCategories(library), query.Categories) {
		return false
	}
	if runtime := strings.TrimSpace(firstNonEmpty(filters.Runtime, query.Environment.Runtime)); runtime != "" && !sameText(runtime, libraryRuntime(library)) {
		return false
	}
	if filters.NetworkAllowed != nil && !*filters.NetworkAllowed && libraryUsesNetwork(library) {
		return false
	}
	if len(filters.AllowedStatuses) > 0 && !containsLifecycle(filters.AllowedStatuses, library.LifecycleStatus) {
		return false
	}
	if len(filters.AllowedTrustLevels) > 0 && !containsTrust(filters.AllowedTrustLevels, library.TrustLevel) {
		return false
	}
	if filters.RequireHealthyDeps && (len(library.Dependencies) > 0 || len(library.Manifest.Dependencies) > 0) {
		return false
	}
	if filters.RequirePolicyAllowed && (library.LifecycleStatus == domain.LifecycleQuarantined || library.LifecycleStatus == domain.LifecycleDeleted) {
		return false
	}
	return true
}

func libraryCategories(library domain.Library) string {
	return strings.Join([]string{library.Category, library.Manifest.Category}, " ")
}

func libraryRuntime(library domain.Library) string {
	return strings.TrimSpace(library.Manifest.Runtime)
}

func libraryUsesNetwork(library domain.Library) bool {
	return library.Manifest.Permissions.Network == domain.NetworkPolicyRestricted || library.Manifest.Permissions.Network == domain.NetworkPolicyFull
}

func libraryLimit(query domain.RetrievalQuery) int {
	if query.Limit > 0 {
		return query.Limit
	}
	if query.FilterTopK > 0 {
		return query.FilterTopK
	}
	return defaultLibraryLimit
}

func limitLibraryCandidates(candidates []domain.LibraryCandidate, limit int) []domain.LibraryCandidate {
	if limit <= 0 || len(candidates) <= limit {
		return candidates
	}
	return candidates[:limit]
}
