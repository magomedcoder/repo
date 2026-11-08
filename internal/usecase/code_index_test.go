package usecase

import "testing"

func TestTokenizeCodeQuery(t *testing.T) {
	cases := map[string][]string{
		"hello world": {"hello", "world"},
		`say "hi"`:    {"say", "hi"},
		"  foo  ":     {"foo"},
		"":            nil,
		"*bad:stuff":  {"bad", "stuff"},
	}
	for in, want := range cases {
		got := tokenizeCodeQuery(in)
		if len(got) != len(want) {
			t.Fatalf("%q: got %#v want %#v", in, got, want)
		}

		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%q: got %#v want %#v", in, got, want)
			}
		}
	}
}

func TestCodeCursorRoundTrip(t *testing.T) {
	cur := encodeCodeCursor(42, "src/main.go")
	id, path, err := decodeCodeCursor(cur)
	if err != nil || id != 42 || path != "src/main.go" {
		t.Fatalf("got %d %q err=%v", id, path, err)
	}

	if _, _, err := decodeCodeCursor("%%%"); err == nil {
		t.Fatal("expected invalid cursor error")
	}
}

func TestExtractLineHits(t *testing.T) {
	body := "alpha\nhello world\nHELLO again\nother\n"
	hits := extractLineHits("f.go", body, "hello", 10)
	if len(hits) != 2 || hits[0].Line != 2 || hits[1].Line != 3 {
		t.Fatalf("%#v", hits)
	}
}
