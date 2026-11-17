package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/project/domain"
)

// stubRepo answers only the reads these tests exercise; embedding the
// interface makes any other call panic, which is what we want here.
type stubRepo struct {
	domain.ProjectRepository
	rows       []*domain.ProjectWithRelations
	lastFilter domain.ProjectFilter
}

func (s *stubRepo) GetAll(_ context.Context, f domain.ProjectFilter) ([]*domain.ProjectWithRelations, error) {
	s.lastFilter = f
	return s.rows, nil
}
func (s *stubRepo) Count(_ context.Context, f domain.ProjectFilter) (int, error) {
	s.lastFilter = f
	return len(s.rows), nil
}
func (s *stubRepo) GetBySlug(_ context.Context, slug string) (*domain.ProjectWithRelations, error) {
	for _, r := range s.rows {
		if r.Slug() == slug {
			return r, nil
		}
	}
	return nil, nil
}

type fakeVerifier struct{}

func (fakeVerifier) Verify(token string) (httpserver.Claims, error) {
	switch token {
	case "admin":
		return httpserver.Claims{UserID: "u1", Role: "admin"}, nil
	case "visitor-role":
		return httpserver.Claims{UserID: "u2", Role: "nobody"}, nil
	}
	return httpserver.Claims{}, errors.New("bad token")
}

func project(t *testing.T, slug, client string, published bool, facts domain.ProjectFacts) *domain.ProjectWithRelations {
	t.Helper()
	p, err := domain.NewProject("Title "+slug, slug, "", nil, nil, false, published, nil, nil, 0, nil, client, "", nil, "", "", facts)
	if err != nil {
		t.Fatal(err)
	}
	return &domain.ProjectWithRelations{Project: p}
}

func newRouter(repo *stubRepo) http.Handler {
	r := chi.NewRouter()
	RegisterRoutes(r, NewProjectHandler(repo), fakeVerifier{})
	return r
}

func get(t *testing.T, h http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestVisitorListIsPinnedToPublished(t *testing.T) {
	repo := &stubRepo{}
	h := newRouter(repo)

	// A visitor asking for drafts + deleted rows must not get them.
	get(t, h, "/projects?is_published=false&include_deleted=true", "")
	if repo.lastFilter.IsPublished == nil || !*repo.lastFilter.IsPublished || repo.lastFilter.IncludeDeleted {
		t.Errorf("visitor filter must be forced to published/non-deleted, got %+v", repo.lastFilter)
	}
	// Same for a token that verifies but has no write permission, and for a bad token.
	for _, tok := range []string{"visitor-role", "garbage"} {
		get(t, h, "/projects?include_deleted=true", tok)
		if repo.lastFilter.IncludeDeleted {
			t.Errorf("token %q must not unlock deleted rows", tok)
		}
	}
	// Count is the same data surface.
	get(t, h, "/projects/count?is_published=false", "")
	if repo.lastFilter.IsPublished == nil || !*repo.lastFilter.IsPublished {
		t.Error("count must be pinned too")
	}

	// Staff keep full control of the filter.
	get(t, h, "/projects?is_published=false&include_deleted=true", "admin")
	if repo.lastFilter.IsPublished == nil || *repo.lastFilter.IsPublished || !repo.lastFilter.IncludeDeleted {
		t.Errorf("staff filter must be honoured, got %+v", repo.lastFilter)
	}
}

func TestVisitorCannotReadDraft(t *testing.T) {
	repo := &stubRepo{rows: []*domain.ProjectWithRelations{project(t, "draft", "", false, domain.ProjectFacts{})}}
	h := newRouter(repo)

	if rec := get(t, h, "/projects/slug/draft", ""); rec.Code != http.StatusNotFound {
		t.Errorf("visitor reading a draft: want 404, got %d", rec.Code)
	}
	if rec := get(t, h, "/projects/slug/draft", "admin"); rec.Code != http.StatusOK {
		t.Errorf("staff reading a draft: want 200, got %d", rec.Code)
	}
}

func TestPersonalClientNameMaskedForVisitors(t *testing.T) {
	now := time.Now()
	person := domain.ProjectFacts{ClientKind: strPtr(domain.ClientKindPerson), ClientConsentNote: "Zalo 2026-11-07"}
	consented := domain.ProjectFacts{ClientKind: strPtr(domain.ClientKindPerson), ClientConsentAt: &now, ClientConsentNote: "Zalo 2026-11-07"}
	business := domain.ProjectFacts{ClientKind: strPtr(domain.ClientKindBusiness)}

	repo := &stubRepo{rows: []*domain.ProjectWithRelations{
		project(t, "person", "Mrs. Tham", true, person),
		project(t, "consented", "Mr. Vuong", true, consented),
		project(t, "business", "Cong ty ABC", true, business),
	}}
	h := newRouter(repo)

	want := map[string]string{"person": "", "consented": "Mr. Vuong", "business": "Cong ty ABC"}
	for slug, name := range want {
		var body projectResponse
		rec := get(t, h, "/projects/slug/"+slug, "")
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.ClientName != name {
			t.Errorf("%s: visitor sees client_name %q, want %q", slug, body.ClientName, name)
		}
		if body.Facts.ClientConsentAt != nil || body.Facts.ClientConsentNote != "" {
			t.Errorf("%s: consent bookkeeping must never reach visitors: %+v", slug, body.Facts)
		}
	}

	// Staff see the real value for editing.
	var body projectResponse
	_ = json.Unmarshal(get(t, h, "/projects/slug/person", "admin").Body.Bytes(), &body)
	if body.ClientName != "Mrs. Tham" || body.Facts.ClientConsentNote == "" {
		t.Errorf("staff must see the stored values, got %q / %+v", body.ClientName, body.Facts)
	}
	if len(body.PublishGaps) == 0 {
		t.Error("staff response should carry publish_gaps")
	}
}

func strPtr(s string) *string { return &s }
