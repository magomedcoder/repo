package ssh

import (
	"testing"

	"github.com/magomedcoder/repo/internal/usecase"
)

func TestParseGitSSHCommand(t *testing.T) {
	service, path, err := parseGitSSHCommand(`git-upload-pack 'user/work/api.git'`)
	if err != nil {
		t.Fatal(err)
	}

	if service != usecase.GitUploadPack || path != "user/work/api.git" {
		t.Fatalf("got %s %q", service, path)
	}

	service, path, err = parseGitSSHCommand(`git-receive-pack "/user/hello.git"`)
	if err != nil {
		t.Fatal(err)
	}

	if service != usecase.GitReceivePack || path != "/user/hello.git" {
		t.Fatalf("got %s %q", service, path)
	}
}

func TestSplitRepoPath(t *testing.T) {
	owner, folder, name, err := splitRepoPath("/user/work/backend/api.git")
	if err != nil {
		t.Fatal(err)
	}

	if owner != "user" || folder != "work/backend" || name != "api" {
		t.Fatalf("%s %s %s", owner, folder, name)
	}

	owner, folder, name, err = splitRepoPath("user/hello")
	if err != nil {
		t.Fatal(err)
	}

	if owner != "user" || folder != "" || name != "hello" {
		t.Fatalf("%s %q %s", owner, folder, name)
	}
}
