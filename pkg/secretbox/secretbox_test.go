package secretbox

import "testing"

func TestSealOpenRoundTrip(t *testing.T) {
	t.Parallel()

	key, err := DeriveKey("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := Seal(key, "s3cret")
	if err != nil || !IsSealed(sealed) {
		t.Fatalf("seal: %q %v", sealed, err)
	}

	plain, err := Open(key, sealed)
	if err != nil || plain != "s3cret" {
		t.Fatalf("open: %q %v", plain, err)
	}
}

func TestOpenLegacyPlaintext(t *testing.T) {
	t.Parallel()

	key, err := DeriveKey("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}

	plain, err := Open(key, "legacy-password")
	if err != nil || plain != "legacy-password" {
		t.Fatalf("legacy: %q %v", plain, err)
	}
}
