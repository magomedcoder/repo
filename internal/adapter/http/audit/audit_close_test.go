package audit

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type memAccessStore struct {
	mu   sync.Mutex
	logs []domain.AccessLog
}

func (s *memAccessStore) Create(l *domain.AccessLog) error {
	s.mu.Lock()
	s.logs = append(s.logs, *l)
	s.mu.Unlock()

	return nil
}

func (s *memAccessStore) List(domain.AccessLogFilter) ([]domain.AccessLog, int, error) {
	return nil, 0, nil
}

func (s *memAccessStore) DeleteOlderThan(time.Time) error {
	return nil
}

func (s *memAccessStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.logs)
}

func TestCloseFlushesAndStopsWorker(t *testing.T) {
	t.Parallel()
	store := &memAccessStore{}
	l := New(store)

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.0.2.1:1"
	l.Login(r, nil)
	l.Close()
	l.Close()

	if store.count() != 1 {
		t.Fatalf("queued event not flushed: %d", store.count())
	}

	l.Login(r, nil)
	if store.count() != 1 {
		t.Fatal("event accepted after Close")
	}
}

func TestVisitIdleGC(t *testing.T) {
	t.Parallel()
	l := &Logger{
		visitLast: map[string]time.Time{"old\n/": time.Now().Add(-2 * accessVisitThrottle)},
		visitIP:   map[string]visitWindow{"old": {start: time.Now().Add(-2 * accessVisitThrottle)}},
	}

	l.visitMu.Lock()
	l.gcVisitsLocked(time.Now())
	l.visitMu.Unlock()

	if len(l.visitLast) != 0 || len(l.visitIP) != 0 {
		t.Fatalf("idle visit entries not collected: %d %d", len(l.visitLast), len(l.visitIP))
	}
}
