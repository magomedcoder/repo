package usecase

import "testing"

func TestValidateMirrorURL(t *testing.T) {
	t.Parallel()

	ok := []string{
		"https://github.com/org/repo.git",
		"http://git.example.com/a/b",
	}
	for _, u := range ok {
		if err := validateMirrorURL(u); err != nil {
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
		"not-a-url",
	}
	for _, u := range bad {
		if err := validateMirrorURL(u); err != ErrInvalidMirrorURL {
			t.Fatalf("%s: want ErrInvalidMirrorURL, got %v", u, err)
		}
	}
}
