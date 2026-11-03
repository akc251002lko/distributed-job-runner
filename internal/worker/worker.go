package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/akc251002lko/distributed-job-runner/internal/job"
	"github.com/akc251002lko/distributed-job-runner/internal/store"
)

type Runner struct { store *store.Store; workerID string; pollInterval time.Duration; leaseDuration time.Duration; logger *slog.Logger }
func New(db *store.Store,id string,poll,lease time.Duration,logger *slog.Logger)*Runner{return &Runner{store:db,workerID:id,pollInterval:poll,leaseDuration:lease,logger:logger}}
func(r *Runner) Run(ctx context.Context)error{
	ticker:=time.NewTicker(r.pollInterval);defer ticker.Stop();r.logger.Info("worker started","worker_id",r.workerID)
	for{
		if err:=r.processOne(ctx);err!=nil&&!errors.Is(err,context.Canceled){r.logger.Error("job processing failed","error",err)}
		select{case<-ctx.Done():r.logger.Info("worker stopping","worker_id",r.workerID);return nil;case<-ticker.C:}
	}
}
func(r *Runner) processOne(ctx context.Context)error{
	claimCtx,cancel:=context.WithTimeout(ctx,5*time.Second);defer cancel()
	j,err:=r.store.Claim(claimCtx,r.workerID,r.leaseDuration);if err!=nil{return fmt.Errorf("claim job: %w",err)}
	if j.ID==""{return nil}
	r.logger.Info("job claimed","job_id",j.ID,"job_type",j.Type,"attempt",j.Attempts)
	if err:=execute(j);err!=nil{
		delay:=retryDelay(j.Attempts);if failErr:=r.store.Fail(ctx,j.ID,j.LeaseToken,err.Error(),delay);failErr!=nil{return fmt.Errorf("record job failure: %w",failErr)}
		r.logger.Warn("job failed","job_id",j.ID,"error",err,"retry_after",delay);return nil
	}
	if err:=r.store.Complete(ctx,j.ID,j.LeaseToken);err!=nil{return fmt.Errorf("complete job: %w",err)}
	r.logger.Info("job completed","job_id",j.ID);return nil
}
func execute(j job.Job)error{switch j.Type{case "noop","echo":return nil;default:return fmt.Errorf("unsupported job type %q",j.Type)}}
func retryDelay(attempt int)time.Duration{
	if attempt<1{attempt=1};delay:=time.Second*time.Duration(1<<min(attempt-1,8))
	if delay>5*time.Minute{delay=5*time.Minute};return delay+time.Duration(rand.Int63n(int64(delay/4+1)))
}
