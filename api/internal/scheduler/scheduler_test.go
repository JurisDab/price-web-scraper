package scheduler

import "testing"

func TestDomainOf(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://www.varle.lt/ispardavimas/", "www.varle.lt"},
		{"https://www.skytech.lt/akcijos.html", "www.skytech.lt"},
		{"https://baitukas.lt/index.php?route=product/selection&type=171", "baitukas.lt"},
		// url.Parse is lenient: text with no scheme/host still "parses"
		// successfully, just with an empty Hostname() rather than an
		// error — so this falls through to an empty string, not the
		// rawURL fallback (that only triggers on an actual parse error,
		// e.g. invalid percent-encoding).
		{"not a url at all", ""},
		{"http://%zz/invalid-escape", "http://%zz/invalid-escape"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := domainOf(tt.url); got != tt.want {
				t.Errorf("domainOf(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestDomainOfGroupsSameSiteTogether(t *testing.T) {
	a := domainOf("https://www.varle.lt/product-a.html")
	b := domainOf("https://www.varle.lt/product-b.html")
	if a != b {
		t.Errorf("two varle.lt URLs resolved to different domains: %q vs %q", a, b)
	}
}
