package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/akc251002lko/distributed-job-runner/internal/observability"
	"github.com/akc251002lko/distributed-job-runner/internal/store"
)

type Server struct { store *store.Store; logger *slog.Logger; token string }
type enqueueRequest struct {
	Type string `json:"type"`
	Payload json.RawMessage `json:"payload"`
	IdempotencyKey string `json:"idempotency_key"`
	MaxAttempts int `json:"max_attempts"`
	RunAt *time.Time `json:"run_at"`
}
func New(db *store.Store, logger *slog.Logger, token string) http.Handler {
	s:=&Server{store:db,logger:logger,token:token}; mux:=http.NewServeMux()
	mux.HandleFunc("GET /healthz",s.health); mux.HandleFunc("GET /readyz",s.ready); mux.Handle("GET /metrics",observability.Handler())
	mux.Handle("POST /v1/jobs",s.auth(http.HandlerFunc(s.enqueue)))
	mux.Handle("GET /v1/jobs/{id}",s.auth(http.HandlerFunc(s.getJob)))
	return requestLog(logger,mux)
}
func(s *Server) auth(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
	if s.token==""{next.ServeHTTP(w,r);return}
	provided:=strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer ")
	if subtle.ConstantTimeCompare([]byte(provided),[]byte(s.token))!=1{writeError(w,http.StatusUnauthorized,"unauthorized");return}
	next.ServeHTTP(w,r)
})}
func(s *Server) health(w http.ResponseWriter,_ *http.Request){writeJSON(w,http.StatusOK,map[string]string{"status":"ok"})}
func(s *Server) ready(w http.ResponseWriter,r *http.Request){if err:=s.store.Ping(r.Context());err!=nil{writeError(w,http.StatusServiceUnavailable,"database unavailable");return};writeJSON(w,http.StatusOK,map[string]string{"status":"ready"})}
func(s *Server) enqueue(w http.ResponseWriter,r *http.Request){
	r.Body=http.MaxBytesReader(w,r.Body,1<<20); dec:=json.NewDecoder(r.Body);dec.DisallowUnknownFields()
	var in enqueueRequest
	if err:=dec.Decode(&in);err!=nil{writeError(w,http.StatusBadRequest,"invalid JSON request");return}
	if err:=dec.Decode(&struct{}{});!errors.Is(err,io.EOF){writeError(w,http.StatusBadRequest,"request must contain one JSON object");return}
	if strings.TrimSpace(in.Type)==""||len(in.Type)>128{writeError(w,http.StatusBadRequest,"type is required and must be at most 128 characters");return}
	if len(in.Payload)==0{in.Payload=json.RawMessage("{}")}
	if !json.Valid(in.Payload){writeError(w,http.StatusBadRequest,"payload must be valid JSON");return}
	if in.MaxAttempts==0{in.MaxAttempts=5};if in.MaxAttempts<1||in.MaxAttempts>100{writeError(w,http.StatusBadRequest,"max_attempts must be between 1 and 100");return}
	runAt:=time.Now().UTC();if in.RunAt!=nil{runAt=in.RunAt.UTC()}
	key:=in.IdempotencyKey;if key==""{key=r.Header.Get("Idempotency-Key")}
	if len(key)>255{writeError(w,http.StatusBadRequest,"idempotency key must be at most 255 characters");return}
	result,err:=s.store.Enqueue(r.Context(),in.Type,in.Payload,key,in.MaxAttempts,runAt)
	if errors.Is(err,store.ErrIdempotencyConflict){writeError(w,http.StatusConflict,"idempotency key was already used with different job data");return}
	if err!=nil{s.logger.Error("enqueue failed","error",err);writeError(w,http.StatusInternalServerError,"could not enqueue job");return}
	observability.Default.JobsEnqueued.Add(1)
	writeJSON(w,http.StatusAccepted,result)
}
func(s *Server) getJob(w http.ResponseWriter,r *http.Request){
	result,err:=s.store.Get(r.Context(),r.PathValue("id"))
	if errors.Is(err,store.ErrNotFound){writeError(w,http.StatusNotFound,"job not found");return}
	if err!=nil{s.logger.Error("get job failed","error",err);writeError(w,http.StatusInternalServerError,"could not retrieve job");return}
	writeJSON(w,http.StatusOK,result)
}
func writeJSON(w http.ResponseWriter,status int,value any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(value)}
func writeError(w http.ResponseWriter,status int,message string){writeJSON(w,status,map[string]string{"error":message})}
func requestLog(logger *slog.Logger,next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){start:=time.Now();observability.Default.HTTPRequests.Add(1);next.ServeHTTP(w,r);logger.Info("http request","method",r.Method,"path",r.URL.Path,"duration",time.Since(start))})}
