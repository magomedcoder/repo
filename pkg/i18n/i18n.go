package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/magomedcoder/repo/langs"
)

type Locale string

const (
	EN Locale = "en"
	RU Locale = "ru"
)

type ctxKey struct{}

var (
	once     sync.Once
	loadErr  error
	messages map[Locale]map[string]string
)

func load() {
	messages = make(map[Locale]map[string]string, 2)
	for _, locale := range []Locale{EN, RU} {
		raw, err := langs.FS.ReadFile(fmt.Sprintf("%s/api.json", locale))
		if err != nil {
			loadErr = err
			return
		}

		var m map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			loadErr = err
			return
		}

		messages[locale] = m
	}
}

func ensureLoaded() {
	once.Do(load)
	if loadErr != nil {
		panic("i18n: load locales: " + loadErr.Error())
	}
}

func ParseAcceptLanguage(header string) Locale {
	header = strings.ToLower(strings.TrimSpace(header))
	if header == "" {
		return EN
	}

	for part := range strings.SplitSeq(header, ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if tag == "ru" || strings.HasPrefix(tag, "ru-") {
			return RU
		}

		if tag == "en" || strings.HasPrefix(tag, "en-") {
			return EN
		}
	}

	return EN
}

func WithLocale(ctx context.Context, locale Locale) context.Context {
	return context.WithValue(ctx, ctxKey{}, locale)
}

func FromContext(ctx context.Context) Locale {
	if locale, ok := ctx.Value(ctxKey{}).(Locale); ok && locale != "" {
		return locale
	}

	return EN
}

func T(ctx context.Context, key string) string {
	ensureLoaded()
	locale := FromContext(ctx)
	if msg, ok := messages[locale][key]; ok {
		return msg
	}

	if msg, ok := messages[EN][key]; ok {
		return msg
	}

	return key
}

func Middleware(next http.Handler) http.Handler {
	ensureLoaded()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := ParseAcceptLanguage(r.Header.Get("Accept-Language"))
		next.ServeHTTP(w, r.WithContext(WithLocale(r.Context(), locale)))
	})
}
