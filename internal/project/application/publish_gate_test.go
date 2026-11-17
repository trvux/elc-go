package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/project/domain"
)

func intp(v int) *int { return &v }

// completeProjectInput is a project that satisfies every publish-gate rule.
func completeProjectInput() domain.CreateProjectInput {
	typeID := "type-1"
	done := time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC)
	return domain.CreateProjectInput{
		Title: "Villa Tan Phu", Slug: "villa-tan-phu", IsPublished: true,
		ProjectTypeID: &typeID,
		CompletedAt:   &done,
		ServiceIDs:    []string{"svc-1"},
		Categories:    []domain.CategoryCondition{{CategoryID: "cat-1", Condition: "new"}},
		Images: []domain.ImageAsset{
			{URL: "https://x/1.webp", Alt: "Dàn nóng VRV"},
			{URL: "https://x/2.webp", Alt: "Dàn lạnh giấu trần"},
			{URL: "https://x/3.webp", Alt: "Thiết bị khí tươi"},
		},
		Facts: domain.ProjectFacts{
			Province: "TP. Hồ Chí Minh", CompletedPrecision: domain.PrecisionDay,
			Brands: []string{"Daikin"}, IndoorUnits: intp(6),
		},
	}
}

func publishGaps(t *testing.T, err error) []string {
	t.Helper()
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	return appErr.Fields["publish"]
}

func TestCreateProject_PublishedMustBeComplete(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	if _, err := CreateProject(ctx, repo, completeProjectInput()); err != nil {
		t.Fatalf("complete project should publish, got %v", err)
	}

	bare := domain.CreateProjectInput{Title: "A", Slug: "a", IsPublished: true}
	_, err := CreateProject(ctx, repo, bare)
	if len(publishGaps(t, err)) < 6 {
		t.Errorf("a bare published project should report many gaps, got %v", err)
	}
}

func TestCreateProject_DraftNeverGated(t *testing.T) {
	repo := newFakeProjectRepository()
	if _, err := CreateProject(context.Background(), repo, domain.CreateProjectInput{Title: "A", Slug: "a"}); err != nil {
		t.Fatalf("draft must save with nothing filled in, got %v", err)
	}
}

func TestToggleProjectPublish_Gated(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	draft, _ := CreateProject(ctx, repo, domain.CreateProjectInput{Title: "A", Slug: "a"})
	if gaps := publishGaps(t, ToggleProjectPublish(ctx, repo, draft.ID(), true)); len(gaps) == 0 {
		t.Error("expected gaps")
	}
	got, _ := GetProjectByID(ctx, repo, draft.ID())
	if got.IsPublished() {
		t.Error("project must stay a draft when the gate fails")
	}
	// Unpublishing is never gated.
	if err := ToggleProjectPublish(ctx, repo, draft.ID(), false); err != nil {
		t.Errorf("unpublish must always work, got %v", err)
	}
}

// An already-published, incomplete row (like 66/71 live ones) must stay
// editable: the gate is only for the draft -> published transition.
func TestUpdateProject_PublishedIncompleteStaysEditable(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	// Seed a published-but-incomplete row straight into the repo, as the live
	// data is (it predates the gate).
	seed, _ := domain.NewProject("Old post", "old-post", "", nil, nil, false, true, nil, nil, 0, nil, "", "", nil, "", "", domain.ProjectFacts{})
	created, _ := repo.Create(ctx, seed, nil, nil)

	newTitle := "Old post (edited)"
	if _, err := UpdateProject(ctx, repo, domain.UpdateProjectInput{ID: created.ID(), Title: &newTitle}); err != nil {
		t.Fatalf("editing an incomplete published row must not be blocked, got %v", err)
	}
}

func TestUpdateProject_DraftToPublishedGated(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	draft, _ := CreateProject(ctx, repo, domain.CreateProjectInput{Title: "A", Slug: "a"})
	yes := true
	_, err := UpdateProject(ctx, repo, domain.UpdateProjectInput{ID: draft.ID(), IsPublished: &yes})
	if len(publishGaps(t, err)) == 0 {
		t.Error("draft -> published with missing facts must be rejected")
	}
}

func TestUpdateProject_FactsReplacedAsAWhole(t *testing.T) {
	repo := newFakeProjectRepository()
	ctx := context.Background()

	in := completeProjectInput()
	in.IsPublished = false
	created, _ := CreateProject(ctx, repo, in)

	facts := domain.ProjectFacts{Province: "Đồng Nai", Brands: []string{"Menred"}}
	got, err := UpdateProject(ctx, repo, domain.UpdateProjectInput{ID: created.ID(), Facts: &facts})
	if err != nil {
		t.Fatalf("update facts: %v", err)
	}
	if got.Facts().IndoorUnits != nil {
		t.Error("omitted number must be cleared (replace-as-a-whole), not kept")
	}
	if got.Facts().CompletedPrecision != domain.PrecisionUnknown {
		t.Errorf("empty precision should default to unknown, got %q", got.Facts().CompletedPrecision)
	}
}

func TestCreateProject_FactsValidation(t *testing.T) {
	repo := newFakeProjectRepository()
	bad := 0
	for name, f := range map[string]domain.ProjectFacts{
		"zero units":    {IndoorUnits: &bad},
		"bad precision": {CompletedPrecision: "week"},
		"bad kind":      {ClientKind: strp("alien")},
	} {
		_, err := CreateProject(context.Background(), repo, domain.CreateProjectInput{Title: "A", Slug: "a", Facts: f})
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
			t.Errorf("%s: expected VALIDATION_ERROR, got %v", name, err)
		}
	}
}

func TestCreateProject_ImageRoleValidated(t *testing.T) {
	repo := newFakeProjectRepository()
	_, err := CreateProject(context.Background(), repo, domain.CreateProjectInput{
		Title: "A", Slug: "a", Images: []domain.ImageAsset{{URL: "https://x/1", Role: "sideways"}},
	})
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || len(appErr.Fields["images"]) == 0 {
		t.Fatalf("expected images field error, got %v", err)
	}
}

func strp(s string) *string { return &s }
