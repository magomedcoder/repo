package git

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/transport"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	xssh "github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/go-git/go-git/v6/storage/memory"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/fsutil"
	"magomedcoder.ru/repo/pkg/ssrf"
	"magomedcoder.ru/repo/pkg/urlredact"
)

const (
	mirrorRemoteName  = "origin"
	defaultMirrorSpec = "refs/heads/*:refs/heads/*"
	sizeCheckEvery    = 2 * time.Second
	mirrorRefsCap     = 200
)

func insecureTLS() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true}
}

func orZeroOptions(opts *domain.MirrorOptions) *domain.MirrorOptions {
	if opts == nil {
		return &domain.MirrorOptions{}
	}

	return opts
}

func mirrorFlags(o *domain.MirrorOptions) (force, prune bool) {
	force, prune = true, false
	if o.Force != nil {
		force = *o.Force
	}

	if o.Prune != nil {
		prune = *o.Prune
	}

	return force, prune
}

func buildSpecs(o *domain.MirrorOptions, force bool) ([]config.RefSpec, error) {
	raw := o.RefSpecs
	if len(raw) == 0 {
		raw = []string{defaultMirrorSpec}
	}

	out := make([]config.RefSpec, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimPrefix(strings.TrimSpace(s), "+")
		if s == "" {
			continue
		}

		if !strings.Contains(s, ":") {
			s = s + ":" + s
		}

		if force {
			s = "+" + s
		}

		spec := config.RefSpec(s)
		if err := spec.Validate(); err != nil {
			return nil, fmt.Errorf("invalid refspec %q: %w", s, err)
		}

		out = append(out, spec)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no refspecs")
	}

	return out, nil
}

func isSSHURL(raw string) bool {
	if strings.HasPrefix(raw, "ssh://") {
		return true
	}

	return !strings.Contains(raw, "://") && strings.Contains(raw, "@") && strings.Contains(raw, ":")
}

func sshUser(raw string, auth *domain.MirrorAuth) string {
	if auth != nil && auth.Username != "" {
		return auth.Username
	}

	if strings.HasPrefix(raw, "ssh://") {
		if u, err := url.Parse(raw); err == nil && u.User != nil && u.User.Username() != "" {
			return u.User.Username()
		}
	} else if at := strings.Index(raw, "@"); at > 0 {
		return raw[:at]
	}

	return "git"
}

func assertMirrorRemote(ctx context.Context, rawURL string) error {
	if err := ssrf.AssertGitRemoteAllowed(ctx, rawURL, nil); err != nil {
		return fmt.Errorf("mirror remote blocked: %w", err)
	}

	return nil
}

func clientOptions(rawURL string, o *domain.MirrorOptions) ([]client.Option, error) {
	o = orZeroOptions(o)
	var out []client.Option

	if isSSHURL(rawURL) {
		if strings.TrimSpace(o.SSHPrivateKey) == "" {
			return nil, fmt.Errorf("ssh mirror requires a private key (ssh-agent is disabled)")
		}

		keys, err := xssh.NewPublicKeys(sshUser(rawURL, o.Auth), []byte(o.SSHPrivateKey), o.SSHPassphrase)
		if err != nil {
			return nil, fmt.Errorf("ssh key: %w", err)
		}

		out = append(out, client.WithSSHAuth(keys))
	} else if o.Auth != nil && (o.Auth.Username != "" || o.Auth.Password != "") {
		user := o.Auth.Username
		if user == "" {
			user = "git"
		}

		out = append(out, client.WithHTTPAuth(&githttp.BasicAuth{Username: user, Password: o.Auth.Password}))
	}

	if o.InsecureTLS {
		out = append(out, client.WithInsecureSkipTLS())
	}

	dial := ssrf.SafeDialContext(o.PinnedHosts, nil)
	out = append(out, client.WithDialer(dial))

	if !isSSHURL(rawURL) {
		tr := &http.Transport{DialContext: dial, ForceAttemptHTTP2: true}
		if o.InsecureTLS {
			tr.TLSClientConfig = insecureTLS()
		}

		httpClient := &http.Client{
			Transport: tr,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return mirrorCheckRedirect(req, via, nil)
			},
		}
		out = append(out, client.WithHTTPClient(httpClient))
	}

	return out, nil
}

func mirrorCheckRedirect(req *http.Request, via []*http.Request, lookup ssrf.LookupIPFunc) error {
	if len(via) >= 10 {
		return errors.New("too many redirects")
	}

	if req.URL == nil {
		return errors.New("redirect missing url")
	}

	scheme := strings.ToLower(req.URL.Scheme)
	if scheme != "https" && scheme != "ssh" {
		return errors.New("insecure redirect scheme")
	}

	if len(via) > 0 && via[len(via)-1].URL != nil {
		prevURL := via[len(via)-1].URL
		prev := strings.ToLower(prevURL.Scheme)
		if prev == "https" && scheme != "https" {
			return errors.New("https to insecure redirect forbidden")
		}

		prevHost := strings.ToLower(prevURL.Hostname())
		nextHost := strings.ToLower(req.URL.Hostname())
		if prevHost != "" && nextHost != "" && prevHost != nextHost {
			return errors.New("cross-host redirect forbidden")
		}
	}

	if err := ssrf.HostAllowed(req.Context(), req.URL.Hostname(), lookup); err != nil {
		return err
	}

	return nil
}

type redactedError struct {
	msg string
	err error
}

func (e *redactedError) Error() string {
	return e.msg
}

func (e *redactedError) Unwrap() error {
	return e.err
}

func redactErr(err error, rawURL string, o *domain.MirrorOptions) error {
	if err == nil {
		return nil
	}

	msg := err.Error()
	if rawURL != "" {
		msg = strings.ReplaceAll(msg, rawURL, urlredact.URL(rawURL))
	}

	if o != nil && o.Auth != nil && o.Auth.Password != "" {
		msg = strings.ReplaceAll(msg, o.Auth.Password, "***")
	}

	if o != nil && o.SSHPassphrase != "" {
		msg = strings.ReplaceAll(msg, o.SSHPassphrase, "***")
	}

	if msg == err.Error() {
		return err
	}

	return &redactedError{
		msg: msg,
		err: err,
	}
}

func isDivergedErr(err error) bool {
	if err == nil {
		return false
	}

	m := strings.ToLower(err.Error())

	return strings.Contains(m, "non-fast-forward") || strings.Contains(m, "fetch first") || strings.Contains(m, "stale info")
}

func ensureMirrorRemote(repo *gogit.Repository, rawURL string) error {
	if rem, err := repo.Remote(mirrorRemoteName); err == nil {
		urls := rem.Config().URLs
		if len(urls) > 0 && urls[0] == rawURL {
			return nil
		}

		_ = repo.DeleteRemote(mirrorRemoteName)
	}

	cfg := &config.RemoteConfig{
		Name:  mirrorRemoteName,
		URLs:  []string{rawURL},
		Fetch: []config.RefSpec{"+" + defaultMirrorSpec},
	}

	if _, err := repo.CreateRemote(cfg); err != nil {
		return fmt.Errorf("mirror remote: %w", err)
	}

	return nil
}

func listRemote(ctx context.Context, rawURL string, co []client.Option) ([]*plumbing.Reference, bool, error) {
	rem := gogit.NewRemote(memory.NewStorage(), &config.RemoteConfig{Name: mirrorRemoteName, URLs: []string{rawURL}})

	refs, err := rem.ListContext(ctx, &gogit.ListOptions{ClientOptions: co})
	if errors.Is(err, transport.ErrEmptyRemoteRepository) {
		return nil, true, nil
	}

	if err != nil {
		return nil, false, err
	}

	real := 0
	for _, r := range refs {
		if r.Name() != plumbing.HEAD {
			real++
		}
	}

	return refs, real == 0, nil
}

func remoteHeadBranch(refs []*plumbing.Reference) string {
	var head *plumbing.Reference
	for _, r := range refs {
		if r.Name() == plumbing.HEAD {
			head = r
			break
		}
	}

	if head == nil {
		return ""
	}

	if head.Type() == plumbing.SymbolicReference {
		return strings.TrimPrefix(head.Target().String(), "refs/heads/")
	}

	var cands []string
	for _, r := range refs {
		if r.Name().IsBranch() && r.Hash() == head.Hash() {
			cands = append(cands, r.Name().Short())
		}
	}

	sort.Strings(cands)
	for _, pref := range []string{"main", "master"} {
		for _, c := range cands {
			if c == pref {
				return c
			}
		}
	}

	if len(cands) > 0 {
		return cands[0]
	}

	return ""
}

func remoteHeadHash(refs []*plumbing.Reference, branch string) string {
	for _, r := range refs {
		if r.Name() == plumbing.HEAD && r.Type() == plumbing.HashReference {
			return r.Hash().String()
		}
	}

	for _, r := range refs {
		if r.Name() == plumbing.NewBranchReferenceName(branch) {
			return r.Hash().String()
		}
	}

	return ""
}

func (r *Repository) MirrorCheck(ctx context.Context, rawURL string, opts *domain.MirrorOptions) (*domain.MirrorRemoteInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("mirror url required")
	}

	if err := assertMirrorRemote(ctx, rawURL); err != nil {
		return nil, err
	}

	o := orZeroOptions(opts)
	co, err := clientOptions(rawURL, o)
	if err != nil {
		return nil, redactErr(err, rawURL, o)
	}

	refs, _, err := listRemote(ctx, rawURL, co)
	if err != nil {
		return nil, redactErr(fmt.Errorf("mirror check: %w", err), rawURL, o)
	}

	return remoteInfoFromRefs(refs), nil
}

func remoteInfoFromRefs(refs []*plumbing.Reference) *domain.MirrorRemoteInfo {
	info := &domain.MirrorRemoteInfo{DefaultBranch: remoteHeadBranch(refs)}
	info.Head = remoteHeadHash(refs, info.DefaultBranch)

	for _, ref := range refs {
		n := ref.Name()
		if !n.IsBranch() && !n.IsTag() {
			continue
		}

		info.Refs = append(info.Refs, domain.RefInfo{
			Name:      n.String(),
			CommitSHA: ref.Hash().String(),
		})
		if len(info.Refs) >= mirrorRefsCap {
			break
		}
	}

	sort.Slice(info.Refs, func(i, j int) bool {
		return info.Refs[i].Name < info.Refs[j].Name
	})

	return info
}

func (r *Repository) MirrorDryRunFetch(ctx context.Context, path, rawURL string, opts *domain.MirrorOptions) (*domain.MirrorRemoteInfo, []domain.RefInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	var err error
	path, err = r.confine(path)
	if err != nil {
		return nil, nil, err
	}

	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, nil, fmt.Errorf("mirror url required")
	}

	if err := assertMirrorRemote(ctx, rawURL); err != nil {
		return nil, nil, err
	}

	o := orZeroOptions(opts)
	force, _ := mirrorFlags(o)
	specs, err := buildSpecs(o, force)
	if err != nil {
		return nil, nil, err
	}

	co, err := clientOptions(rawURL, o)
	if err != nil {
		return nil, nil, redactErr(err, rawURL, o)
	}

	refs, _, err := listRemote(ctx, rawURL, co)
	if err != nil {
		return nil, nil, redactErr(fmt.Errorf("mirror dry-run: %w", err), rawURL, o)
	}

	info := remoteInfoFromRefs(refs)
	local := localRefHashes(path)
	var out []domain.RefInfo
	for _, ref := range refs {
		if ref.Name() == plumbing.HEAD || ref.Type() != plumbing.HashReference {
			continue
		}

		for _, spec := range specs {
			if !spec.Match(ref.Name()) {
				continue
			}

			dst := spec.Dst(ref.Name())
			if local[dst.String()] != ref.Hash().String() {
				out = append(out, domain.RefInfo{
					Name:      dst.String(),
					CommitSHA: ref.Hash().String(),
				})
			}

			break
		}

		if len(out) >= mirrorRefsCap {
			break
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	if len(out) > mirrorRefsCap {
		out = out[:mirrorRefsCap]
	}

	return info, out, nil
}

func (r *Repository) MirrorVerify(ctx context.Context, path, rawURL, direction string, opts *domain.MirrorOptions) (*domain.MirrorVerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var err error
	path, err = r.confine(path)
	if err != nil {
		return nil, err
	}

	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("mirror url required")
	}

	if err := assertMirrorRemote(ctx, rawURL); err != nil {
		return nil, err
	}

	o := orZeroOptions(opts)
	force, prune := mirrorFlags(o)
	specs, err := buildSpecs(o, force)
	if err != nil {
		return nil, err
	}

	co, err := clientOptions(rawURL, o)
	if err != nil {
		return nil, redactErr(err, rawURL, o)
	}

	refs, empty, err := listRemote(ctx, rawURL, co)
	if err != nil {
		return nil, redactErr(fmt.Errorf("mirror verify: %w", err), rawURL, o)
	}

	defaultBranch := remoteHeadBranch(refs)
	res := &domain.MirrorVerifyResult{
		OK:            true,
		DefaultBranch: defaultBranch,
		Head:          remoteHeadHash(refs, defaultBranch),
	}

	local := localRefHashes(path)
	remote := map[string]string{}
	if !empty {
		for _, ref := range refs {
			if ref.Name() == plumbing.HEAD || ref.Type() != plumbing.HashReference {
				continue
			}

			if !ref.Name().IsBranch() && !ref.Name().IsTag() {
				continue
			}

			remote[ref.Name().String()] = ref.Hash().String()
		}
	}

	var diffs []domain.MirrorRefDiff
	checked := 0

	switch direction {
	case domain.MirrorPush:
		for name, sha := range local {
			rn := plumbing.ReferenceName(name)
			if !rn.IsBranch() && !rn.IsTag() {
				continue
			}

			for _, spec := range specs {
				if !spec.Match(rn) {
					continue
				}

				dst := spec.Dst(rn).String()
				checked++
				remoteSHA, ok := remote[dst]
				if !ok {
					diffs = append(diffs, domain.MirrorRefDiff{
						Name:   dst,
						Local:  sha,
						Reason: "missing_remote",
					})
					break
				}

				if remoteSHA != sha {
					diffs = append(diffs, domain.MirrorRefDiff{
						Name:   dst,
						Local:  sha,
						Remote: remoteSHA,
						Reason: "mismatch",
					})
				}

				break
			}
		}
	default:
		expected := map[string]string{}
		for name, sha := range remote {
			rn := plumbing.ReferenceName(name)
			for _, spec := range specs {
				if !spec.Match(rn) {
					continue
				}

				expected[spec.Dst(rn).String()] = sha
				break
			}
		}

		for name, want := range expected {
			checked++
			got, ok := local[name]
			if !ok {
				diffs = append(diffs, domain.MirrorRefDiff{
					Name:   name,
					Remote: want,
					Reason: "missing_local",
				})
				continue
			}

			if got != want {
				diffs = append(diffs, domain.MirrorRefDiff{
					Name:   name,
					Local:  got,
					Remote: want,
					Reason: "mismatch",
				})
			}
		}

		if prune {
			for name, sha := range local {
				rn := plumbing.ReferenceName(name)
				if !rn.IsBranch() && !rn.IsTag() {
					continue
				}

				if _, ok := expected[name]; ok {
					continue
				}

				if !refCoveredByDest(specs, rn) {
					continue
				}

				diffs = append(diffs, domain.MirrorRefDiff{
					Name:   name,
					Local:  sha,
					Reason: "extra_local",
				})
			}
		}
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Name < diffs[j].Name
	})
	res.Checked = checked
	res.TotalDiffs = len(diffs)
	res.OK = len(diffs) == 0
	if len(diffs) > mirrorRefsCap {
		res.Diffs = diffs[:mirrorRefsCap]
		res.Truncated = true
	} else {
		res.Diffs = diffs
	}

	return res, nil
}

func refCoveredByDest(specs []config.RefSpec, name plumbing.ReferenceName) bool {
	local := name.String()
	for _, spec := range specs {
		if destPatternCovers(refSpecDestPattern(spec), local) {
			return true
		}
	}
	return false
}

func refSpecDestPattern(spec config.RefSpec) string {
	s := strings.TrimPrefix(string(spec), "+")
	if i := strings.LastIndex(s, ":"); i >= 0 {
		return s[i+1:]
	}

	return s
}

func destPatternCovers(pattern, name string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}

	if !strings.Contains(pattern, "*") {
		return pattern == name
	}

	parts := strings.SplitN(pattern, "*", 2)
	if len(parts) != 2 {
		return false
	}

	return strings.HasPrefix(name, parts[0]) && strings.HasSuffix(name, parts[1]) && len(name) >= len(parts[0])+len(parts[1])
}

func (r *Repository) confine(path string) (string, error) {
	return fsutil.SafePath(r.Root, path)
}

func localRefHashes(path string) map[string]string {
	out := map[string]string{}

	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return out
	}

	defer repo.Close()

	iter, err := repo.References()
	if err != nil {
		return out
	}

	defer iter.Close()

	_ = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.HashReference {
			out[ref.Name().String()] = ref.Hash().String()
		}

		return nil
	})

	return out
}

func (r *Repository) MirrorClone(ctx context.Context, path, rawURL string, opts *domain.MirrorOptions) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	var err error
	path, err = r.confine(path)
	if err != nil {
		return "", err
	}

	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("mirror url required")
	}

	if err := assertMirrorRemote(ctx, rawURL); err != nil {
		return "", err
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return "", fmt.Errorf("repository already exists: %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create parent directory: %w", err)
	}

	o := orZeroOptions(opts)
	if o.Partial {
		return "", domain.ErrMirrorPartialUnavailable
	}

	if o.LFS {
		return "", domain.ErrMirrorLFSUnavailable
	}

	repo, err := gogit.PlainInit(path, true)
	if err != nil {
		_ = os.RemoveAll(path)
		return "", fmt.Errorf("init mirror: %w", err)
	}

	head, err := r.mirrorFetchInto(ctx, repo, path, rawURL, o)
	_ = repo.Close()

	if err != nil {
		_ = os.RemoveAll(path)
		return "", err
	}

	if err := os.Chmod(path, 0o700); err != nil {
		_ = os.RemoveAll(path)
		return "", err
	}

	if head == "" {
		_ = os.WriteFile(filepath.Join(path, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)
	}

	return head, nil
}

func (r *Repository) MirrorFetch(ctx context.Context, path, rawURL string, opts *domain.MirrorOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var err error
	path, err = r.confine(path)
	if err != nil {
		return err
	}

	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("mirror url required")
	}

	if err := assertMirrorRemote(ctx, rawURL); err != nil {
		return err
	}

	defer r.LockRepo(path)()

	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("open mirror: %w", err)
	}
	defer repo.Close()

	o := orZeroOptions(opts)
	if o.Partial {
		return domain.ErrMirrorPartialUnavailable
	}

	if o.LFS {
		return domain.ErrMirrorLFSUnavailable
	}

	if _, err := r.mirrorFetchInto(ctx, repo, path, rawURL, o); err != nil {
		return err
	}

	return nil
}

func (r *Repository) mirrorFetchInto(ctx context.Context, repo *gogit.Repository, path, rawURL string, o *domain.MirrorOptions) (string, error) {
	force, prune := mirrorFlags(o)
	specs, err := buildSpecs(o, force)
	if err != nil {
		return "", err
	}

	if err := ensureMirrorRemote(repo, rawURL); err != nil {
		return "", err
	}

	co, err := clientOptions(rawURL, o)
	if err != nil {
		return "", redactErr(err, rawURL, o)
	}

	gctx, done := guardSize(ctx, path, o.MaxBytes)
	refs, empty, err := listRemote(gctx, rawURL, co)
	if err != nil {
		_ = done()
		return "", redactErr(fmt.Errorf("mirror fetch: %w", err), rawURL, o)
	}

	if empty {
		return "", done()
	}

	fo := &gogit.FetchOptions{
		RemoteName:    mirrorRemoteName,
		RemoteURL:     rawURL,
		RefSpecs:      specs,
		Force:         force,
		Prune:         prune,
		Tags:          gogit.NoTags,
		ClientOptions: co,
	}

	if o.Progress != nil {
		fo.Progress = o.Progress
	}

	err = repo.FetchContext(gctx, fo)
	if sizeErr := done(); sizeErr != nil {
		return "", sizeErr
	}

	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) && !errors.Is(err, transport.ErrEmptyRemoteRepository) {
		if isDivergedErr(err) {
			err = fmt.Errorf("%w: %v", domain.ErrMirrorDiverged, err)
		}

		return "", redactErr(fmt.Errorf("mirror fetch: %w", err), rawURL, o)
	}

	head := remoteHeadBranch(refs)
	if head != "" {
		name := plumbing.NewBranchReferenceName(head)
		if _, rerr := repo.Storer.Reference(name); rerr == nil {
			_ = repo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, name))
		} else {
			head = ""
		}
	}

	return head, nil
}

func (r *Repository) MirrorPush(ctx context.Context, path, rawURL string, opts *domain.MirrorOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var err error
	path, err = r.confine(path)
	if err != nil {
		return err
	}

	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("mirror url required")
	}

	if err := assertMirrorRemote(ctx, rawURL); err != nil {
		return err
	}

	defer r.LockRepo(path)()

	o := orZeroOptions(opts)
	force, prune := mirrorFlags(o)
	specs, err := buildSpecs(o, force)
	if err != nil {
		return err
	}

	co, err := clientOptions(rawURL, o)
	if err != nil {
		return redactErr(err, rawURL, o)
	}

	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("open mirror: %w", err)
	}
	defer repo.Close()

	if err := ensureMirrorRemote(repo, rawURL); err != nil {
		return err
	}

	if o.LFS {
		return domain.ErrMirrorLFSUnavailable
	}

	po := &gogit.PushOptions{
		RemoteName:    mirrorRemoteName,
		RemoteURL:     rawURL,
		RefSpecs:      specs,
		Force:         force,
		Prune:         prune,
		ClientOptions: co,
	}

	if o.Progress != nil {
		po.Progress = o.Progress
	}

	err = repo.PushContext(ctx, po)
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		if isDivergedErr(err) {
			err = fmt.Errorf("%w: %v", domain.ErrMirrorDiverged, err)
		}

		return redactErr(fmt.Errorf("mirror push: %w", err), rawURL, o)
	}

	return nil
}

func dirSize(path string) int64 {
	var total int64

	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if info, ierr := d.Info(); ierr == nil {
			total += info.Size()
		}

		return nil
	})

	return total
}

func repoSizeBytes(path string) int64 {
	objects := filepath.Join(path, "objects")
	if st, err := os.Stat(objects); err == nil && st.IsDir() {
		return dirSize(objects)
	}

	return dirSize(path)
}

func guardSize(ctx context.Context, path string, max int64) (context.Context, func() error) {
	if max <= 0 {
		return ctx, func() error { return nil }
	}

	gctx, cancel := context.WithCancelCause(ctx)
	stop := make(chan struct{})
	var exceeded atomic.Bool

	go func() {
		t := time.NewTicker(sizeCheckEvery)
		defer t.Stop()

		for {
			select {
			case <-stop:
				return
			case <-gctx.Done():
				return
			case <-t.C:
				if repoSizeBytes(path) > max {
					exceeded.Store(true)
					cancel(domain.ErrMirrorTooLarge)

					return
				}
			}
		}
	}()

	return gctx, func() error {
		close(stop)
		defer cancel(nil)

		if exceeded.Load() || repoSizeBytes(path) > max {
			return fmt.Errorf("%w (limit %d bytes)", domain.ErrMirrorTooLarge, max)
		}

		return nil
	}
}
