package usecase

import "testing"

func TestNormalizePaginationRejectsUnlimited(t *testing.T) {
	offset, limit := NormalizePagination(-5, 0)
	if offset != 0 || limit != 50 {
		t.Fatalf("zero/neg limit => default, got offset=%d limit=%d", offset, limit)
	}

	offset, limit = NormalizePagination(0, 1000)
	if offset != 0 || limit != 100 {
		t.Fatalf("cap max limit, got offset=%d limit=%d", offset, limit)
	}

	offset, limit = NormalizePagination(99999, 10)
	if offset != 10000 || limit != 10 {
		t.Fatalf("cap offset, got offset=%d limit=%d", offset, limit)
	}
}
