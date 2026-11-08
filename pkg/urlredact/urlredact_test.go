package urlredact

import "testing"

func TestURL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                                            "",
		"https://u:p@example.com/a/b.git":             "https://example.com/a/b.git",
		"https://example.com/a.git?token=s3cret":      "https://example.com/a.git",
		"https://example.com/a.git#frag":              "https://example.com/a.git",
		"https://u:p@example.com/a.git?x=1#f":         "https://example.com/a.git",
		"https://example.com/a.git?":                  "https://example.com/a.git",
		"git@example.com:team/repo.git":               "git@example.com:team/repo.git",
		"git@example.com:team/repo.git?token=s3cret":  "git@example.com:team/repo.git",
		"https://u:p@exa mple.com/a.git?token=s3cret": "https://exa mple.com/a.git",
	}
	for in, want := range cases {
		if got := URL(in); got != want {
			t.Errorf("URL(%q) = %q, want %q", in, got, want)
		}
	}
}
