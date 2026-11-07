package metrics

import (
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

type Result int

const (
	ResultOK Result = iota
	ResultFail
	ResultSkipped
	resultCount
)

func (r Result) String() string {
	switch r {
	case ResultOK:
		return "ok"
	case ResultFail:
		return "fail"
	default:
		return "skipped"
	}
}

var directions = [...]string{"pull", "push", "other"}

var durationBuckets = [...]float64{0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600, 1800}

var lagBuckets = [...]float64{60, 300, 900, 1800, 3600, 10800, 86400}

const lagThresholdSeconds = 300 // 5m

type Metrics struct {
	total      [len(directions)][resultCount]atomic.Int64
	buckets    [len(durationBuckets) + 1]atomic.Int64
	durCount   atomic.Int64
	durSumNano atomic.Int64

	lagBuckets [len(lagBuckets) + 1]atomic.Int64
	lagCount   atomic.Int64
	lagSumNano atomic.Int64
	lagOver5m  atomic.Int64
}

func New() *Metrics {
	return &Metrics{}
}

func dirIndex(direction string) int {
	for i, d := range directions {
		if d == direction {
			return i
		}
	}

	return len(directions) - 1
}

func (m *Metrics) IncSync(direction string, r Result) {
	if m == nil || r < 0 || r >= resultCount {
		return
	}

	m.total[dirIndex(direction)][r].Add(1)
}

func (m *Metrics) SyncTotal(r Result) int64 {
	if m == nil || r < 0 || r >= resultCount {
		return 0
	}

	var n int64
	for i := range directions {
		n += m.total[i][r].Load()
	}

	return n
}

func (m *Metrics) SyncDurationObserve(d time.Duration) {
	if m == nil {
		return
	}

	sec := d.Seconds()
	idx := len(durationBuckets)
	for i, le := range durationBuckets {
		if sec <= le {
			idx = i
			break
		}
	}

	m.buckets[idx].Add(1)
	m.durCount.Add(1)
	m.durSumNano.Add(d.Nanoseconds())
}

func (m *Metrics) DurationCount() int64 {
	if m == nil {
		return 0
	}

	return m.durCount.Load()
}

func (m *Metrics) ObserveSync(direction string, r Result, d time.Duration) {
	m.IncSync(direction, r)
	if r != ResultSkipped {
		m.SyncDurationObserve(d)
	}
}

func (m *Metrics) SetLagSeconds(repoID uint, seconds float64) {
	_ = repoID
	m.ObserveLag(seconds)
}

func (m *Metrics) ObserveLag(seconds float64) {
	if m == nil || seconds < 0 {
		return
	}

	idx := len(lagBuckets)
	for i, le := range lagBuckets {
		if seconds <= le {
			idx = i
			break
		}
	}

	m.lagBuckets[idx].Add(1)
	m.lagCount.Add(1)
	m.lagSumNano.Add(int64(seconds * 1e9))

	if seconds > lagThresholdSeconds {
		m.lagOver5m.Add(1)
	}
}

func (m *Metrics) ForgetRepo(repoID uint) {
	_ = repoID
}

func (m *Metrics) LagOverThreshold() int64 {
	if m == nil {
		return 0
	}

	return m.lagOver5m.Load()
}

func (m *Metrics) LagCount() int64 {
	if m == nil {
		return 0
	}

	return m.lagCount.Load()
}

func (m *Metrics) WriteText(w io.Writer) error {
	if m == nil {
		return nil
	}

	if _, err := fmt.Fprintln(w, "# HELP repo_mirror_sync_total Mirror sync attempts by direction and result."); err != nil {
		return err
	}
	fmt.Fprintln(w, "# TYPE repo_mirror_sync_total counter")
	for i, d := range directions {
		for r := range resultCount {
			fmt.Fprintf(w, "repo_mirror_sync_total{direction=%q,result=%q} %d\n", d, r.String(), m.total[i][r].Load())
		}
	}

	fmt.Fprintln(w, "# HELP repo_mirror_sync_duration_seconds Mirror sync duration.")
	fmt.Fprintln(w, "# TYPE repo_mirror_sync_duration_seconds histogram")
	var cum int64
	for i, le := range durationBuckets {
		cum += m.buckets[i].Load()
		fmt.Fprintf(w, "repo_mirror_sync_duration_seconds_bucket{le=\"%g\"} %d\n", le, cum)
	}

	cum += m.buckets[len(durationBuckets)].Load()
	fmt.Fprintf(w, "repo_mirror_sync_duration_seconds_bucket{le=\"+Inf\"} %d\n", cum)
	fmt.Fprintf(w, "repo_mirror_sync_duration_seconds_sum %g\n", float64(m.durSumNano.Load())/1e9)
	fmt.Fprintf(w, "repo_mirror_sync_duration_seconds_count %d\n", m.durCount.Load())

	fmt.Fprintln(w, "# HELP repo_mirror_lag_seconds Seconds since last successful sync (observed).")
	fmt.Fprintln(w, "# TYPE repo_mirror_lag_seconds histogram")
	cum = 0
	for i, le := range lagBuckets {
		cum += m.lagBuckets[i].Load()
		fmt.Fprintf(w, "repo_mirror_lag_seconds_bucket{le=\"%g\"} %d\n", le, cum)
	}
	cum += m.lagBuckets[len(lagBuckets)].Load()
	fmt.Fprintf(w, "repo_mirror_lag_seconds_bucket{le=\"+Inf\"} %d\n", cum)
	fmt.Fprintf(w, "repo_mirror_lag_seconds_sum %g\n", float64(m.lagSumNano.Load())/1e9)
	fmt.Fprintf(w, "repo_mirror_lag_seconds_count %d\n", m.lagCount.Load())

	fmt.Fprintln(w, "# HELP repo_mirror_lag_over_threshold Count of lag observations above 5 minutes.")
	fmt.Fprintln(w, "# TYPE repo_mirror_lag_over_threshold gauge")
	fmt.Fprintf(w, "repo_mirror_lag_over_threshold %d\n", m.lagOver5m.Load())

	return nil
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_ = m.WriteText(w)
	})
}
