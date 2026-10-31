package git

import "errors"

var (
	ErrEmptyRepository = errors.New("empty repository")
	ErrRefNotFound     = errors.New("ref not found")
	ErrPathNotFound    = errors.New("path not found")
	ErrCommitNotFound  = errors.New("commit not found")
	ErrFileNotFound    = errors.New("file not found")
	ErrReadmeNotFound  = errors.New("readme not found")
	ErrNotDirectory    = errors.New("path is not a directory")
	ErrMergeConflict   = errors.New("merge conflict")
	ErrNotFastForward  = errors.New("not fast-forward")
)
