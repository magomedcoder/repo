package git

import (
	"context"
	"crypto/tls"
	"io"
	"os"
	"strconv"
	"strings"

	"magomedcoder.ru/repo/internal/domain"
)

const lfsOutputCap = 64 << 10

type cappedBuffer struct {
	buf []byte
	cap int
	n   int
}

func newCappedBuffer(cap int) *cappedBuffer {
	return &cappedBuffer{cap: cap}
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	c.n += len(p)
	if c.cap <= 0 {
		return len(p), nil
	}

	if len(p) >= c.cap {
		c.buf = append(c.buf[:0], p[len(p)-c.cap:]...)

		return len(p), nil
	}

	need := len(c.buf) + len(p) - c.cap
	if need > 0 {
		c.buf = c.buf[need:]
	}

	c.buf = append(c.buf, p...)

	return len(p), nil
}

func (c *cappedBuffer) String() string {
	return string(c.buf)
}

var _ io.Writer = (*cappedBuffer)(nil)

func insecureTLS() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true}
}

func lfsRemoteURL(raw string, _ *domain.MirrorOptions) string {
	return raw
}

func lfsEnv(o *domain.MirrorOptions, keyFile string) []string {
	env := filterEnv(os.Environ(), "GIT_TERMINAL_PROMPT", "GIT_SSL_NO_VERIFY", "GIT_SSH_COMMAND")
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	if o != nil && o.InsecureTLS {
		env = append(env, "GIT_SSL_NO_VERIFY=true")
	}

	if keyFile != "" {
		ssh := "ssh -i " + strconv.Quote(keyFile) + " -o IdentitiesOnly=yes -o BatchMode=yes"
		env = append(env, "GIT_SSH_COMMAND="+ssh)
	}

	return env
}

func filterEnv(env []string, dropKeys ...string) []string {
	drop := map[string]struct{}{}
	for _, k := range dropKeys {
		drop[k] = struct{}{}
	}

	out := make([]string, 0, len(env))
	for _, e := range env {
		key, _, _ := strings.Cut(e, "=")
		if _, ok := drop[key]; ok {
			continue
		}

		out = append(out, e)
	}
	return out
}

func lfsRun(ctx context.Context, repoPath, rawURL string, o *domain.MirrorOptions, verb string) error {
	_ = ctx
	_ = repoPath
	_ = rawURL
	_ = o
	_ = verb
	return domain.ErrMirrorLFSUnavailable
}

func partialMirrorClone(ctx context.Context, path, rawURL string, o *domain.MirrorOptions) (string, error) {
	_ = ctx
	_ = path
	_ = rawURL
	_ = o
	return "", domain.ErrMirrorPartialUnavailable
}
