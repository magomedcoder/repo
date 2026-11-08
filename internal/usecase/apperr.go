package usecase

import "magomedcoder.ru/repo/pkg/apperr"

type Kind = apperr.Kind
type Error = apperr.Error

const (
	KindValidation   = apperr.KindValidation
	KindUnauthorized = apperr.KindUnauthorized
	KindForbidden    = apperr.KindForbidden
	KindNotFound     = apperr.KindNotFound
	KindConflict     = apperr.KindConflict
	KindRateLimited  = apperr.KindRateLimited
	KindUnavailable  = apperr.KindUnavailable
)

var ErrBusy = NewError(KindUnavailable, "server_busy")

func NewError(kind Kind, code string) *Error {
	return apperr.New(kind, code)
}
