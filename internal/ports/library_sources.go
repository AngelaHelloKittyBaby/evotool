package ports

import (
	"context"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
)

// LibrarySources resolves exact, approved library versions into sandbox files.
type LibrarySources interface {
	Resolve(ctx context.Context, refs []domain.DependencyRef) ([]domain.SourceFile, []domain.DependencyRef, error)
}

// LibraryLister is optional for stores that support enumeration.
type LibraryLister interface {
	List(ctx context.Context) ([]domain.Library, error)
}
