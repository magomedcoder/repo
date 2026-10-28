package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func SafePath(root, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path outside data root")
	}

	for _, seg := range strings.Split(filepath.ToSlash(path), "/") {
		if seg == ".." {
			return "", fmt.Errorf("path outside data root")
		}
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if root != "" {
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}

		rel, err := filepath.Rel(rootAbs, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("path outside data root")
		}
	}

	cur := abs
	for {
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return "", err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink")
		}

		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}

		cur = parent
	}

	return abs, nil
}
