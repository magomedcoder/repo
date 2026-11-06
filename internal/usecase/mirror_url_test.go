package usecase

import "testing"

func TestValidateMirrorURL(t *testing.T) {
	t.Parallel()

	ok := []string{
		"https://github.com/magomedcoder/repo.git",
		"http://git.example.com/a/b",
		"ssh://git@github.com/magomedcoder/repo.git",
		"git@github.com:org/repo.git",
		"deploy@git.example.com:team/app.git",
	}
	for _, u := range ok {
		if err := (mirrorURLPolicy{}).validate(u); err != nil {
			t.Fatalf("%s: %v", u, err)
		}
	}

	bad := []string{
		"",
		"ftp://example.com/repo.git",
		"file:///tmp/repo.git",
		"https://localhost/repo.git",
		"http://127.0.0.1/repo.git",
		"https://10.0.0.1/repo.git",
		"https://192.168.1.1/x.git",
		"https://user:pass@github.com/magomedcoder/repo.git",
		"ssh://git@127.0.0.1/repo.git",
		"git@10.0.0.5:org/repo.git",
		"http://169.254.169.254/latest",
		"not-a-url",
		"git@host: spaced",
	}
	for _, u := range bad {
		if err := (mirrorURLPolicy{}).validate(u); err != ErrInvalidMirrorURL {
			t.Fatalf("%s: want ErrInvalidMirrorURL, got %v", u, err)
		}
	}
}

func TestParsePinnedHosts(t *testing.T) {
	t.Parallel()

	p := mirrorURLPolicy{}

	got, err := p.parsePinnedHosts("git.example.com=203.0.113.7, Other.Example=198.51.100.1")
	if err != nil || got["git.example.com"] != "203.0.113.7" || got["other.example"] != "198.51.100.1" {
		t.Fatalf("got=%v err=%v", got, err)
	}

	if _, err := p.parsePinnedHosts("git.example.com=10.0.0.1"); err != ErrInvalidMirrorURL {
		t.Fatalf("private pin must be rejected: %v", err)
	}

	if _, err := p.parsePinnedHosts("nonsense"); err != ErrInvalidMirrorOption {
		t.Fatalf("bad pin: %v", err)
	}
}
