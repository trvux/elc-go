package application

import (
	"context"
	"testing"

	"github.com/trvux/elc-go/internal/redirect/domain"
)

func create(t *testing.T, repo *fakeRepository, from, to string) *domain.Redirect {
	t.Helper()
	r, err := CreateRedirect(context.Background(), repo, domain.CreateRedirectInput{FromPath: from, ToPath: to})
	if err != nil {
		t.Fatalf("CreateRedirect(%s -> %s): %v", from, to, err)
	}
	return r
}

func TestCreateRedirect_NormalizesInput(t *testing.T) {
	repo := newFakeRepository()
	r := create(t, repo, "https://dienmayelc.com.vn/dich-vu/cu/", "/dich-vu/moi/")
	if r.FromPath() != "/dich-vu/cu" || r.ToPath() != "/dich-vu/moi" {
		t.Errorf("got %s -> %s", r.FromPath(), r.ToPath())
	}
}

func TestCreateRedirect_RejectsDuplicateSource(t *testing.T) {
	repo := newFakeRepository()
	create(t, repo, "/a", "/b")
	if _, err := CreateRedirect(context.Background(), repo, domain.CreateRedirectInput{FromPath: "/a", ToPath: "/c"}); err == nil {
		t.Error("expected a conflict for a source that already redirects")
	}
}

func TestCreateRedirect_FlattensChainAtTarget(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("/b", "/c")
	r := create(t, repo, "/a", "/b") // /a -> /b -> /c  must become  /a -> /c
	if r.ToPath() != "/c" {
		t.Errorf("target should be flattened to /c, got %s", r.ToPath())
	}
}

func TestCreateRedirect_RepointsEarlierRedirectsToTheNewTarget(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("/old", "/a") // already points at /a
	create(t, repo, "/a", "/b")
	if got := repo.targetOf("/old"); got != "/b" {
		t.Errorf("/old should now go straight to /b, got %q", got)
	}
}

func TestCreateRedirect_RejectsLoop(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("/b", "/a")
	if _, err := CreateRedirect(context.Background(), repo, domain.CreateRedirectInput{FromPath: "/a", ToPath: "/b"}); err == nil {
		t.Error("expected /a -> /b to be rejected: /b already sends visitors back to /a")
	}
}

func TestCreateRedirect_ExternalTargetIsKept(t *testing.T) {
	repo := newFakeRepository()
	r := create(t, repo, "/a", "https://example.com/x")
	if r.ToPath() != "https://example.com/x" {
		t.Errorf("got %s", r.ToPath())
	}
}

func TestCreateRedirect_InactiveRedirectsAreNotFollowed(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("/b", "/c")
	inactive := false
	for id := range repo.items {
		if _, err := UpdateRedirect(context.Background(), repo, domain.UpdateRedirectInput{ID: id, IsActive: &inactive}); err != nil {
			t.Fatal(err)
		}
	}
	r := create(t, repo, "/a", "/b")
	if r.ToPath() != "/b" {
		t.Errorf("a disabled redirect must not be followed: got %s", r.ToPath())
	}
}

func TestUpdateRedirect_Retargets(t *testing.T) {
	repo := newFakeRepository()
	a := create(t, repo, "/a", "/b")
	to := "/c"
	updated, err := UpdateRedirect(context.Background(), repo, domain.UpdateRedirectInput{ID: a.ID(), ToPath: &to})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ToPath() != "/c" {
		t.Errorf("got %s", updated.ToPath())
	}
}

func TestUpdateRedirect_FlattensANewTargetThatItselfRedirects(t *testing.T) {
	repo := newFakeRepository()
	a := create(t, repo, "/a", "/b")
	repo.seed("/x", "/y")
	to := "/x" // /x already redirects to /y, so /a must go straight to /y
	updated, err := UpdateRedirect(context.Background(), repo, domain.UpdateRedirectInput{ID: a.ID(), ToPath: &to})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ToPath() != "/y" {
		t.Errorf("target should be flattened to /y, got %s", updated.ToPath())
	}
}

func TestUpdateRedirect_RejectsLoopAndMissing(t *testing.T) {
	repo := newFakeRepository()
	a := create(t, repo, "/a", "/b")
	create(t, repo, "/b", "/c")
	loop := "/a" // /a -> /a
	if _, err := UpdateRedirect(context.Background(), repo, domain.UpdateRedirectInput{ID: a.ID(), ToPath: &loop}); err == nil {
		t.Error("expected a self-loop to be rejected")
	}
	if _, err := UpdateRedirect(context.Background(), repo, domain.UpdateRedirectInput{ID: "nope"}); err == nil {
		t.Error("expected not-found for an unknown id")
	}
}

func TestDeleteRedirect(t *testing.T) {
	repo := newFakeRepository()
	a := create(t, repo, "/a", "/b")
	if err := DeleteRedirect(context.Background(), repo, a.ID()); err != nil {
		t.Fatal(err)
	}
	if err := DeleteRedirect(context.Background(), repo, a.ID()); err == nil {
		t.Error("deleting twice should report not-found")
	}
}

func TestListRedirects_ClampsPaging(t *testing.T) {
	repo := newFakeRepository()
	if _, _, err := ListRedirects(context.Background(), repo, domain.Filter{Limit: 100000, Offset: -5}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ListRedirects(context.Background(), repo, domain.Filter{Source: "garbage"}); err == nil {
		t.Error("an unknown source filter should be rejected")
	}
}
