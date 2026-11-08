package gitref

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const MaxLen = 255

var ErrInvalid = errors.New("invalid ref name")

func Validate(ref string) error {
	if ref == "" {
		return nil
	}

	if len(ref) > MaxLen {
		return fmt.Errorf("%w: too long", ErrInvalid)
	}

	if strings.Contains(ref, "..") || strings.Contains(ref, "@{") || strings.Contains(ref, "//") ||
		strings.HasPrefix(ref, "-") || strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/") ||
		strings.HasSuffix(ref, ".") || strings.HasSuffix(ref, ".lock") {
		return ErrInvalid
	}

	for _, c := range ref {
		switch {
		case unicode.IsLetter(c), unicode.IsDigit(c):
		case strings.ContainsRune("-_./+@#=,", c):
		default:
			return ErrInvalid
		}
	}

	return nil
}
