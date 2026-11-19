package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
	"path"
	"regexp"
	"sort"
	"strings"
)

// LibrarySources loads dependency-free, approved Python libraries. Transitive
// dependencies are rejected rather than silently omitted from the sandbox.
type LibrarySources struct{ Store ports.LibraryStore }

// Resolve installs modules under shared/<library-name> and pins a source digest.
// If the current version or contents change, a previously validated tool fails closed.
func (s LibrarySources) Resolve(ctx context.Context, refs []domain.DependencyRef) ([]domain.SourceFile, []domain.DependencyRef, error) {
	files := []domain.SourceFile{{Path: "shared/__init__.py"}}
	pins := make([]domain.DependencyRef, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if s.Store == nil {
			return nil, nil, fmt.Errorf("library store required")
		}
		if ref.Kind != "" && ref.Kind != domain.DependencyLibrary || ref.Optional || ref.Version == "" {
			return nil, nil, fmt.Errorf("required exact library version expected")
		}
		id := firstNonEmpty(ref.ID, ref.Name)
		if seen[id] {
			return nil, nil, fmt.Errorf("duplicate library reference %s", id)
		}
		seen[id] = true
		lib, err := s.Store.Get(ctx, id)
		if err != nil {
			return nil, nil, fmt.Errorf("load library %s: %w", id, err)
		}
		if !regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`).MatchString(lib.Name) || seen["name:"+lib.Name] {
			return nil, nil, fmt.Errorf("invalid or duplicate library module name")
		}
		seen["name:"+lib.Name] = true
		if lib.Manifest.Runtime != "python" || lib.Manifest.Name != lib.Name || lib.CurrentVersion != ref.Version || (lib.LifecycleStatus != domain.LifecycleApproved && lib.LifecycleStatus != domain.LifecycleActive) || (lib.TrustLevel != domain.TrustValidated && lib.TrustLevel != domain.TrustReviewed && lib.TrustLevel != domain.TrustTrusted) {
			return nil, nil, fmt.Errorf("library %s is not approved for requested version", id)
		}
		if len(lib.Manifest.Dependencies)+len(lib.Dependencies) > 0 {
			return nil, nil, fmt.Errorf("transitive library dependencies unsupported")
		}
		p := lib.Manifest.Permissions
		if (p.Network != "" && p.Network != domain.NetworkPolicyNone) || len(p.AllowedPaths)+len(p.AllowedCommands)+len(p.Environment) > 0 {
			return nil, nil, fmt.Errorf("privileged library rejected")
		}
		var source []domain.SourceFile
		matches := 0
		for _, v := range lib.Versions {
			if v.Version == ref.Version {
				source = append([]domain.SourceFile{}, v.Files...)
				matches++
			}
		}
		if matches != 1 || len(source) == 0 || len(source) > 100 {
			return nil, nil, fmt.Errorf("library version source missing or ambiguous")
		}
		sort.Slice(source, func(i, j int) bool { return source[i].Path < source[j].Path })
		raw, _ := json.Marshal(struct {
			Manifest domain.LibraryManifest
			Files    []domain.SourceFile
		}{lib.Manifest, source})
		sum := sha256.Sum256(raw)
		digest := hex.EncodeToString(sum[:])
		if ref.Digest != "" && ref.Digest != digest {
			return nil, nil, fmt.Errorf("library %s changed: revalidation required", id)
		}
		module := "shared/" + lib.Name + "/"
		local := map[string]bool{}
		total := 0
		hasInit := false
		for _, f := range source {
			rel := strings.TrimPrefix(f.Path, "source/")
			if rel == "" || rel == "." || path.IsAbs(rel) || path.Clean(rel) != rel || strings.ContainsAny(rel, "\\:") || rel == ".." || strings.HasPrefix(rel, "../") || local[rel] {
				return nil, nil, fmt.Errorf("unsafe library path %q", f.Path)
			}
			local[rel] = true
			total += len(f.Content)
			if total > 524288 {
				return nil, nil, fmt.Errorf("library exceeds source limit")
			}
			if rel == "__init__.py" {
				hasInit = true
			}
			files = append(files, domain.SourceFile{Path: module + rel, Content: f.Content})
		}
		if !hasInit {
			files = append(files, domain.SourceFile{Path: module + "__init__.py"})
		}
		ref.ID = id
		ref.Name = lib.Name
		ref.Kind = domain.DependencyLibrary
		ref.Digest = digest
		pins = append(pins, ref)
	}
	return files, pins, nil
}
