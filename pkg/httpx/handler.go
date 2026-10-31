package httpx

import "slices"

import "net/http"

type Handler func(*Context) error

type Middleware func(Handler) Handler

func Handle(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := &Context{W: w, R: r}
		_ = h(c)
	}
}

func HTTPMiddleware(mw func(http.Handler) http.Handler) Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			var err error
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.W = w
				c.R = r
				err = next(c)
			})).ServeHTTP(c.W, c.R)
			return err
		}
	}
}

func chain(mws []Middleware, h Handler) Handler {
	for _, mw := range slices.Backward(mws) {
		h = mw(h)
	}

	return h
}
