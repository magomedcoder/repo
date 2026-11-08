package apperr

type Kind int

const (
	KindValidation Kind = iota
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindRateLimited
	KindUnavailable
)

type Error struct {
	Kind Kind
	Code string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	return e.Code
}

func New(kind Kind, code string) *Error {
	return &Error{
		Kind: kind,
		Code: code,
	}
}
