package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExposition(t *testing.T) {
	t.Parallel()

	m := New()
	m.ObserveSync("pull", ResultOK, 3*time.Second)
	m.ObserveSync("push", ResultFail, 700*time.Millisecond)
	m.ObserveSync("pull", ResultSkipped, 0)
	m.SetLagSeconds(7, 42)
	m.ObserveLag(400)

	if m.SyncTotal(ResultOK) != 1 || m.SyncTotal(ResultFail) != 1 || m.SyncTotal(ResultSkipped) != 1 {
		t.Fatal("counters")
	}

	if m.DurationCount() != 2 {
		t.Fatalf("skipped must not observe duration: %d", m.DurationCount())
	}

	if m.LagCount() != 2 || m.LagOverThreshold() != 1 {
		t.Fatalf("lag count=%d over=%d", m.LagCount(), m.LagOverThreshold())
	}

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	for _, want := range []string{
		`repo_mirror_sync_total{direction="pull",result="ok"} 1`,
		`repo_mirror_sync_total{direction="push",result="fail"} 1`,
		`repo_mirror_sync_duration_seconds_bucket{le="+Inf"} 2`,
		`repo_mirror_sync_duration_seconds_count 2`,
		`repo_mirror_lag_seconds_count 2`,
		`repo_mirror_lag_over_threshold 1`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in\n%s", want, body)
		}
	}

	if strings.Contains(body, `repo_mirror_lag_seconds{repo_id=`) {
		t.Fatal("must not emit per-repo lag labels")
	}

	m.ForgetRepo(7)
}

func TestNilSafe(t *testing.T) {
	t.Parallel()

	var m *Metrics
	m.ObserveSync("pull", ResultOK, time.Second)
	m.SetLagSeconds(1, 1)
	m.ObserveLag(1)
	_ = m.WriteText(&strings.Builder{})
}
