package observability

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	HTTPRequests atomic.Uint64
	JobsEnqueued atomic.Uint64
	JobsClaimed atomic.Uint64
	JobsSucceeded atomic.Uint64
	JobsFailed atomic.Uint64
	LeaseRenewals atomic.Uint64
	LeaseLosses atomic.Uint64
}

var Default = &Metrics{}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		m := Default
		_, _ = fmt.Fprintf(w,
			"# HELP djr_http_requests_total HTTP requests handled by this process.\n# TYPE djr_http_requests_total counter\ndjr_http_requests_total %d\n"+
				"# HELP djr_jobs_enqueued_total Jobs accepted by this API process.\n# TYPE djr_jobs_enqueued_total counter\ndjr_jobs_enqueued_total %d\n"+
				"# HELP djr_jobs_claimed_total Jobs claimed by this worker process.\n# TYPE djr_jobs_claimed_total counter\ndjr_jobs_claimed_total %d\n"+
				"# HELP djr_jobs_succeeded_total Jobs completed by this worker process.\n# TYPE djr_jobs_succeeded_total counter\ndjr_jobs_succeeded_total %d\n"+
				"# HELP djr_jobs_failed_total Job attempts recorded as failed by this worker process.\n# TYPE djr_jobs_failed_total counter\ndjr_jobs_failed_total %d\n"+
				"# HELP djr_lease_renewals_total Successful lease renewals by this worker process.\n# TYPE djr_lease_renewals_total counter\ndjr_lease_renewals_total %d\n"+
				"# HELP djr_lease_losses_total Lease ownership losses observed by this worker process.\n# TYPE djr_lease_losses_total counter\ndjr_lease_losses_total %d\n",
			m.HTTPRequests.Load(), m.JobsEnqueued.Load(), m.JobsClaimed.Load(),
			m.JobsSucceeded.Load(), m.JobsFailed.Load(), m.LeaseRenewals.Load(), m.LeaseLosses.Load())
	})
}
