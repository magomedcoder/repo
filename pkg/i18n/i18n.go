package i18n

import (
	"context"
	json "encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"magomedcoder.ru/repo/langs"
)

type Locale string

const (
	EN Locale = "en"
	RU Locale = "ru"
)

type ctxKey struct{}

var (
	once         sync.Once
	loadFailOnce sync.Once
	loadErr      error
	messages     map[Locale]map[string]string
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

func Preload() error {
	once.Do(load)
	if loadErr != nil {
		return fmt.Errorf("i18n: load locales: %w", loadErr)
	}

	return nil
}

func ensureLoaded() bool {
	if err := Preload(); err != nil {
		loadFailOnce.Do(func() { slog.Error("i18n unavailable, falling back to message keys", "err", err) })
		return false
	}

	return true
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
	if !ensureLoaded() {
		return key
	}

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
	_ = ensureLoaded()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := ParseAcceptLanguage(r.Header.Get("Accept-Language"))
		next.ServeHTTP(w, r.WithContext(WithLocale(r.Context(), locale)))
	})
}
