package git

import (
	"strconv"
	"strings"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
)

func TestLFSRemoteURLOmitsPassword(t *testing.T) {
	t.Parallel()

	raw := "https://git.example.com/org/app.git"
	got := lfsRemoteURL(raw, &domain.MirrorOptions{
		Auth: &domain.MirrorAuth{
			Username: "bot",
			Password: "s3cret",
		},
	})
	if got != raw || strings.Contains(got, "s3cret") {
		t.Fatalf("must not embed password: %q", got)
	}
}

func TestLFSEnvQuotesSSHKeyPath(t *testing.T) {
	t.Parallel()

	key := `/tmp/key with spaces;rm -rf /`
	env := lfsEnv(&domain.MirrorOptions{}, key)
	var sshCmd string
	for _, e := range env {
		if strings.HasPrefix(e, "GIT_SSH_COMMAND=") {
			sshCmd = e
		}
	}

	if sshCmd == "" {
		t.Fatal("GIT_SSH_COMMAND missing")
	}

	want := "GIT_SSH_COMMAND=ssh -i " + strconv.Quote(key) + " -o IdentitiesOnly=yes -o BatchMode=yes"
	if sshCmd != want {
		t.Fatalf("got %q want %q", sshCmd, want)
	}
}
