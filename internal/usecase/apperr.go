package usecase

type Kind int

const (
	KindValidation Kind = iota
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindRateLimited
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

func NewError(kind Kind, code string) *Error {
	return &Error{
		Kind: kind,
		Code: code,
	}
}
