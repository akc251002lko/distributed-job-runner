package observability

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerExposesPrometheusCounters(t *testing.T) {
	m:=Default
	old:=m.HTTPRequests.Load()
	m.HTTPRequests.Store(old+1)
	t.Cleanup(func(){m.HTTPRequests.Store(old)})
	rec:=httptest.NewRecorder()
	Handler().ServeHTTP(rec,httptest.NewRequest("GET","/metrics",nil))
	if rec.Code!=200{t.Fatalf("status = %d, want 200",rec.Code)}
	body:=rec.Body.String()
	for _,want:=range []string{"# TYPE djr_http_requests_total counter","djr_http_requests_total"}{
		if !strings.Contains(body,want){t.Fatalf("metrics response missing %q",want)}
	}
}
