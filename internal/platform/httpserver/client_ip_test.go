package httpserver

import (
	"net/http"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		forwarded  string
		remoteAddr string
		want       string
	}{
		{
			name:       "single hop (Nginx overwrites, doesn't append)",
			forwarded:  "203.0.113.5",
			remoteAddr: "10.0.0.1:12345",
			want:       "203.0.113.5",
		},
		{
			name:       "multiple hops: trusts the LAST one (Nginx's own append), not the client-suppliable first",
			forwarded:  "1.2.3.4, 203.0.113.5",
			remoteAddr: "10.0.0.1:12345",
			want:       "203.0.113.5",
		},
		{
			name:       "spoofed first hop must not win",
			forwarded:  "9.9.9.9",
			remoteAddr: "10.0.0.1:12345",
			want:       "9.9.9.9", // no proxy hop to strip in this single-entry case
		},
		{
			name:       "no X-Forwarded-For falls back to RemoteAddr",
			forwarded:  "",
			remoteAddr: "203.0.113.9:54321",
			want:       "203.0.113.9",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}, RemoteAddr: c.remoteAddr}
			if c.forwarded != "" {
				r.Header.Set("X-Forwarded-For", c.forwarded)
			}
			got := ClientIP(r)
			if got != c.want {
				t.Errorf("ClientIP() = %q, want %q", got, c.want)
			}
		})
	}
}
