package domain

import "errors"

var ErrNotFound = errors.New("not found")
var ErrBusy = errors.New("server busy")
var ErrConflict = errors.New("conflict")
