package httpx

import (
	"context"
	"net/http"

	json "encoding/json/v2"

	"magomedcoder.ru/repo/pkg/i18n"
)

type Context struct {
	W http.ResponseWriter
	R *http.Request
}

func (c *Context) Ctx() context.Context {
	return c.R.Context()
}

func (c *Context) Param(name string) string {
	return c.R.PathValue(name)
}

func (c *Context) Query(name string) string {
	return c.R.URL.Query().Get(name)
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitzero"`
}

func BindJSON[T any](c *Context) (T, error) {
	var v T
	err := json.UnmarshalRead(c.R.Body, &v)
	return v, err
}

func (c *Context) JSON(status int, v any) error {
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(status)
	return json.MarshalWrite(c.W, v)
}

func OK[T any](c *Context, v T) error {
	return c.JSON(http.StatusOK, v)
}

func Created[T any](c *Context, v T) error {
	return c.JSON(http.StatusCreated, v)
}

func (c *Context) NoContent() error {
	c.W.WriteHeader(http.StatusNoContent)
	return nil
}

func (c *Context) Error(status int, code string) error {
	return c.JSON(status, ErrorResponse{
		Error: i18n.T(c.Ctx(), code),
		Code:  code,
	})
}

func (c *Context) ErrorPlain(status int, message string) error {
	return c.JSON(status, ErrorResponse{
		Error: message,
		Code:  message,
	})
}
