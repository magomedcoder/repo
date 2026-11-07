package git

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/storer"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"magomedcoder.ru/repo/internal/domain"
)

func enforceProtectedBranches(ctx context.Context, info *transport.PreReceiveInfo) error {
	if info == nil {
		return nil
	}

	for _, cmd := range info.Commands {
		if cmd == nil {
			continue
		}

		oldRev := cmd.Old.String()
		newRev := cmd.New.String()
		refname := string(cmd.Name)

		ancestor := true
		if !cmd.Old.IsZero() && !cmd.New.IsZero() {
			ok, err := isAncestorInStorer(ctx, info.Storer, cmd.Old, cmd.New)
			if err != nil || !ok {
				ancestor = false
			}
		}

		if err := domain.CheckRefUpdate(true, refname, oldRev, newRev, ancestor); err != nil {
			return err
		}
	}
	return nil
}

func isAncestorInStorer(ctx context.Context, st storer.EncodedObjectStorer, oldHash, newHash plumbing.Hash) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	oldCommit, err := object.GetCommit(st, oldHash)
	if err != nil {
		return false, fmt.Errorf("old commit: %w", err)
	}

	newCommit, err := object.GetCommit(st, newHash)
	if err != nil {
		return false, fmt.Errorf("new commit: %w", err)
	}

	return oldCommit.IsAncestor(newCommit)
}
