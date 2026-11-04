package api
import("errors";"net/http";"net/http/httptest";"strings";"testing";"github.com/akc251002lko/distributed-job-runner/internal/store")
func TestAuthRejectsInvalidBearerAndAllowsConfiguredClient(t *testing.T){s:=&Server{apiKeys:map[string]string{"mobile":"0123456789abcdef0123456789abcdef"},limiter:newRateLimiter(10,10)};next:=s.auth(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.Context().Value(clientKey)!="mobile"{t.Error("client identity missing")};w.WriteHeader(http.StatusNoContent)}));bad:=httptest.NewRecorder();next.ServeHTTP(bad,httptest.NewRequest(http.MethodGet,"/",nil));if bad.Code!=http.StatusUnauthorized{t.Fatalf("status = %d, want 401",bad.Code)};good:=httptest.NewRequest(http.MethodGet,"/",nil);good.Header.Set("Authorization","Bearer 0123456789abcdef0123456789abcdef");ok:=httptest.NewRecorder();next.ServeHTTP(ok,good);if ok.Code!=http.StatusNoContent{t.Fatalf("status = %d, want 204",ok.Code)}}
func TestTraceMiddlewareCreatesRequestID(t *testing.T){next:=traceRequests(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if len(r.Context().Value(traceKey).(string))!=32{t.Error("trace ID missing")}}));rec:=httptest.NewRecorder();next.ServeHTTP(rec,httptest.NewRequest(http.MethodGet,"/",nil));if rec.Header().Get("X-Request-ID")==""||rec.Header().Get("traceparent")==""{t.Fatal("trace headers missing")}}


func TestEnqueueRejectsWhitespacePaddedType(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/v1/jobs", strings.NewReader(`{"type":" noop "}`))
	rec := httptest.NewRecorder()
	s.enqueue(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestListErrorClassification(t *testing.T) {
	if got := listErrorStatus(store.ErrInvalidCursor); got != http.StatusBadRequest {
		t.Fatalf("invalid cursor status = %d, want 400", got)
	}
	if got := listErrorStatus(errors.New("database unavailable")); got != http.StatusInternalServerError {
		t.Fatalf("database error status = %d, want 500", got)
	}
}
