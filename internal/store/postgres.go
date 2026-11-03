package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/akc251002lko/distributed-job-runner/internal/job"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("job not found")
type Store struct { pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*Store,error) {
	cfg,err:=pgxpool.ParseConfig(url); if err!=nil { return nil,fmt.Errorf("parse database config: %w",err) }
	cfg.MaxConns=20; cfg.MinConns=2
	p,err:=pgxpool.NewWithConfig(ctx,cfg); if err!=nil { return nil,fmt.Errorf("connect postgres: %w",err) }
	if err=p.Ping(ctx); err!=nil { p.Close(); return nil,fmt.Errorf("ping postgres: %w",err) }
	return &Store{pool:p},nil
}
func(s *Store) Close(){s.pool.Close()}
func(s *Store) Ping(ctx context.Context)error{return s.pool.Ping(ctx)}

func(s *Store) Enqueue(ctx context.Context, kind string, payload json.RawMessage, key string, max int, runAt time.Time)(job.Job,error){
	id,err:=newID(); if err!=nil{return job.Job{},err}
	q:=`INSERT INTO jobs(id,job_type,payload,idempotency_key,status,max_attempts,run_at)
	VALUES($1,$2,$3,NULLIF($4,''),'pending',$5,$6)
	ON CONFLICT(idempotency_key) WHERE idempotency_key IS NOT NULL
	DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key
	RETURNING id,job_type,payload,status,attempts,max_attempts,run_at,created_at,updated_at,lease_expires_at,COALESCE(last_error,''),COALESCE(worker_id,'')`
	return scan(s.pool.QueryRow(ctx,q,id,kind,[]byte(payload),key,max,runAt))
}
func(s *Store) Get(ctx context.Context,id string)(job.Job,error){
	q:=`SELECT id,job_type,payload,status,attempts,max_attempts,run_at,created_at,updated_at,lease_expires_at,COALESCE(last_error,''),COALESCE(worker_id,'') FROM jobs WHERE id=$1`
	j,err:=scan(s.pool.QueryRow(ctx,q,id)); if errors.Is(err,pgx.ErrNoRows){return job.Job{},ErrNotFound}; return j,err
}
func(s *Store) Claim(ctx context.Context,worker string,lease time.Duration)(job.Job,error){
	token,err:=newID(); if err!=nil{return job.Job{},err}
	q:=`WITH candidate AS (SELECT id FROM jobs WHERE (status='pending' AND run_at<=now()) OR (status='running' AND lease_expires_at<now()) ORDER BY run_at,created_at FOR UPDATE SKIP LOCKED LIMIT 1)
	UPDATE jobs j SET status='running',attempts=j.attempts+1,worker_id=$1,lease_token=$2,lease_expires_at=now()+$3::interval,updated_at=now()
	FROM candidate WHERE j.id=candidate.id
	RETURNING j.id,j.job_type,j.payload,j.status,j.attempts,j.max_attempts,j.run_at,j.created_at,j.updated_at,j.lease_expires_at,COALESCE(j.last_error,''),COALESCE(j.worker_id,''),j.lease_token`
	var j job.Job
	err=s.pool.QueryRow(ctx,q,worker,token,lease.String()).Scan(&j.ID,&j.Type,&j.Payload,&j.Status,&j.Attempts,&j.MaxAttempts,&j.RunAt,&j.CreatedAt,&j.UpdatedAt,&j.LeaseExpiresAt,&j.LastError,&j.WorkerID,&j.LeaseToken)
	if errors.Is(err,pgx.ErrNoRows){return job.Job{},nil}; return j,err
}
func(s *Store) Complete(ctx context.Context,id,token string)error{
	tag,err:=s.pool.Exec(ctx,`UPDATE jobs SET status='succeeded',completed_at=now(),lease_token=NULL,lease_expires_at=NULL,worker_id=NULL,updated_at=now() WHERE id=$1 AND status='running' AND lease_token=$2`,id,token)
	if err!=nil{return err}; if tag.RowsAffected()!=1{return errors.New("job lease lost before completion")}; return nil
}
func(s *Store) Fail(ctx context.Context,id,token,message string,delay time.Duration)error{
	tag,err:=s.pool.Exec(ctx,`UPDATE jobs SET status=CASE WHEN attempts>=max_attempts THEN 'dead' ELSE 'pending' END,
	run_at=CASE WHEN attempts>=max_attempts THEN run_at ELSE now()+$4::interval END,last_error=left($3,4000),
	lease_token=NULL,lease_expires_at=NULL,worker_id=NULL,updated_at=now()
	WHERE id=$1 AND status='running' AND lease_token=$2`,id,token,message,delay.String())
	if err!=nil{return err}; if tag.RowsAffected()!=1{return errors.New("job lease lost before failure update")}; return nil
}
func scan(row pgx.Row)(job.Job,error){
	var j job.Job
	err:=row.Scan(&j.ID,&j.Type,&j.Payload,&j.Status,&j.Attempts,&j.MaxAttempts,&j.RunAt,&j.CreatedAt,&j.UpdatedAt,&j.LeaseExpiresAt,&j.LastError,&j.WorkerID)
	return j,err
}
func newID()(string,error){var b [16]byte;if _,err:=rand.Read(b[:]);err!=nil{return "",fmt.Errorf("generate ID: %w",err)};return hex.EncodeToString(b[:]),nil}
