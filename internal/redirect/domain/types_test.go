package domain

import "testing"

func TestNormalizePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/dich-vu/ten-cu", "/dich-vu/ten-cu"},
		{"/dich-vu/ten-cu/", "/dich-vu/ten-cu"},
		{"  /dich-vu/ten-cu  ", "/dich-vu/ten-cu"},
		{"//dich-vu///ten-cu//", "/dich-vu/ten-cu"},
		{"/dich-vu/ten-cu?utm=1#top", "/dich-vu/ten-cu"},
		{"https://dienmayelc.com.vn/dich-vu/ten-cu/", "/dich-vu/ten-cu"},
		{"https://www.dienmayelc.com.vn/dich-vu/ten-cu?x=1", "/dich-vu/ten-cu"},
		{"/san-pham/m%C3%A1y-l%E1%BA%A1nh", "/san-pham/máy-lạnh"}, // decoded: nginx matches the decoded $uri
	}
	for _, c := range cases {
		got, err := NormalizePath(c.in)
		if err != nil {
			t.Errorf("NormalizePath(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizePath_Rejects(t *testing.T) {
	for _, in := range []string{"", "   ", "dich-vu/khong-co-slash", "/co khoang trang", "https://example.com/x"} {
		if _, err := NormalizePath(in); err == nil {
			t.Errorf("NormalizePath(%q): expected an error", in)
		}
	}
}

func TestNormalizeTarget_AllowsExternalHTTPSOnly(t *testing.T) {
	got, err := NormalizeTarget("https://example.com/page")
	if err != nil || got != "https://example.com/page" {
		t.Errorf("external https target: got %q, %v", got, err)
	}
	if _, err := NormalizeTarget("http://example.com/page"); err == nil {
		t.Error("external http target should be rejected")
	}
	got, err = NormalizeTarget("https://dienmayelc.com.vn/dich-vu/moi/")
	if err != nil || got != "/dich-vu/moi" {
		t.Errorf("same-site URL should reduce to a path: got %q, %v", got, err)
	}
}

func TestNewRedirect_Rules(t *testing.T) {
	if _, err := NewRedirect("/a", "/b", "", SourceManual); err != nil {
		t.Fatalf("valid redirect rejected: %v", err)
	}
	bad := []struct{ name, from, to string }{
		{"same path", "/a", "/a/"},
		{"homepage as source", "/", "/b"},
		{"admin as source", "/admin/users", "/b"},
		{"api as source", "/api/x", "/b"},
		{"next assets as source", "/_next/static/x", "/b"},
		{"empty target", "/a", ""},
	}
	for _, c := range bad {
		if _, err := NewRedirect(c.from, c.to, "", SourceManual); err == nil {
			t.Errorf("%s: expected a validation error", c.name)
		}
	}
	r, _ := NewRedirect("/a", "/b", "  ghi chú  ", SourceManual)
	if r.Note() != "ghi chú" || !r.IsActive() || r.Source() != SourceManual {
		t.Errorf("unexpected defaults: note=%q active=%v source=%s", r.Note(), r.IsActive(), r.Source())
	}
}
