package git

func withGitTimeout[T any](fn func() (T, error)) (T, error) {
	return fn()
}
