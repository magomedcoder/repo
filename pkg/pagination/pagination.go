package pagination

const (
	DefaultLimit = 50
	MaxLimit     = 100
	MaxOffset    = 10000
)

func Normalize(offset, limit int) (int, int) {
	return Clamp(offset, limit, DefaultLimit, MaxLimit)
}

func Clamp(offset, limit, defLimit, maxLimit int) (int, int) {
	if defLimit <= 0 {
		defLimit = DefaultLimit
	}

	if maxLimit <= 0 {
		maxLimit = MaxLimit
	}

	if limit <= 0 {
		limit = defLimit
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	if offset < 0 {
		offset = 0
	}

	if offset > MaxOffset {
		offset = MaxOffset
	}

	return offset, limit
}
