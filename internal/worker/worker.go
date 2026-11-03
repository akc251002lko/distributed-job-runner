package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/akc251002lko/distributed-job-runner/internal/job"
	"github.com/akc251002lko/distributed-job-runner/internal/observability"
	"github.com/akc251002lko/distributed-job-runner/internal/store"
)

type Runner struct {
	store *store.Store
	workerID string
	pollInterval time.Duration
	leaseDuration time.Duration
	logger *slog.Logger
}

func New(db *store.Store,id string,poll,lease time.Duration,logger *slog.Logger)*Runner {
	return &Runner{store:db,workerID:id,pollInterval:poll,leaseDuration:lease,logger:logger}
}

func(r *Runner) Run(ctx context.Context)error {
	ticker:=time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	r.logger.Info("worker started","worker_id",r.workerID)
	for {
		if err:=r.processOne(ctx);err!=nil&&!errors.Is(err,context.Canceled) {
			r.logger.Error("job processing failed","error",err)
		}
		select {
		case <-ctx.Done():
			r.logger.Info("worker stopping","worker_id",r.workerID)
			return nil
		case <-ticker.C:
		}
	}
}

func(r *Runner) processOne(ctx context.Context)error {
	claimCtx,cancelClaim:=context.WithTimeout(ctx,5*time.Second)
	claimed,err:=r.store.Claim(claimCtx,r.workerID,r.leaseDuration)
	cancelClaim()
	if err!=nil{return fmt.Errorf("claim job: %w",err)}
	if claimed.ID==""{return nil}
	observability.Default.JobsClaimed.Add(1)
	r.logger.Info("job claimed","job_id",claimed.ID,"job_type",claimed.Type,"attempt",claimed.Attempts)

	jobCtx,cancelJob:=context.WithCancel(ctx)
	heartbeatErr:=make(chan error,1)
	heartbeatDone:=make(chan struct{})
	go r.maintainLease(jobCtx,cancelJob,claimed,heartbeatErr,heartbeatDone)

	execErr:=execute(jobCtx,claimed)
	cancelJob()
	<-heartbeatDone

	select {
	case leaseErr:=<-heartbeatErr:
		observability.Default.LeaseLosses.Add(1)
		r.logger.Warn("job lease lost; abandoning execution","job_id",claimed.ID,"error",leaseErr)
		return nil
	default:
	}
	if ctx.Err()!=nil {
		// Leave the lease to expire so another worker can safely recover it.
		return nil
	}
	if execErr!=nil {
		delay:=retryDelay(claimed.Attempts)
		if failErr:=r.store.Fail(ctx,claimed.ID,claimed.LeaseToken,execErr.Error(),delay);failErr!=nil {
			if errors.Is(failErr,context.Canceled){return nil}
			return fmt.Errorf("record job failure: %w",failErr)
		}
		observability.Default.JobsFailed.Add(1)
		r.logger.Warn("job failed","job_id",claimed.ID,"error",execErr,"retry_after",delay)
		return nil
	}
	if err:=r.store.Complete(ctx,claimed.ID,claimed.LeaseToken);err!=nil {
		if errors.Is(err,context.Canceled){return nil}
		return fmt.Errorf("complete job: %w",err)
	}
	observability.Default.JobsSucceeded.Add(1)
	r.logger.Info("job completed","job_id",claimed.ID)
	return nil
}

func(r *Runner) maintainLease(ctx context.Context,cancel context.CancelFunc,j job.Job,errCh chan<- error,done chan<- struct{}) {
	defer close(done)
	interval:=r.leaseDuration/3
	if interval<=0{interval=time.Second}
	ticker:=time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			renewCtx,cancelRenew:=context.WithTimeout(ctx,5*time.Second)
			err:=r.store.RenewLease(renewCtx,j.ID,j.LeaseToken,r.leaseDuration)
			cancelRenew()
			if err!=nil {
				if ctx.Err()==nil {
					errCh<-err
					cancel()
				}
				return
			}
			observability.Default.LeaseRenewals.Add(1)
		}
	}
}

func execute(ctx context.Context,j job.Job)error {
	if err:=ctx.Err();err!=nil{return err}
	switch j.Type {
	case "noop","echo":
		return nil
	default:
		return fmt.Errorf("unsupported job type %q",j.Type)
	}
}

func retryDelay(attempt int)time.Duration {
	if attempt<1{attempt=1}
	delay:=time.Second*time.Duration(1<<min(attempt-1,8))
	if delay>5*time.Minute{delay=5*time.Minute}
	return delay+time.Duration(rand.Int63n(int64(delay/4+1)))
}
