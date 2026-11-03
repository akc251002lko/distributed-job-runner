package worker

import (
	"context"
	"testing"
	"time"

	"github.com/akc251002lko/distributed-job-runner/internal/job"
)

func TestExecuteSupportedAndUnsupportedTypes(t *testing.T) {
	ctx:=context.Background()
	if err:=execute(ctx,job.Job{Type:"noop"});err!=nil{t.Fatalf("noop: %v",err)}
	if err:=execute(ctx,job.Job{Type:"echo"});err!=nil{t.Fatalf("echo: %v",err)}
	if err:=execute(ctx,job.Job{Type:"unregistered"});err==nil{t.Fatal("expected unsupported job type to fail")}
}
func TestExecuteHonorsCancellation(t *testing.T) {
	ctx,cancel:=context.WithCancel(context.Background());cancel()
	if err:=execute(ctx,job.Job{Type:"noop"});err==nil{t.Fatal("expected canceled context to stop execution")}
}
func TestRetryDelayIsBoundedWithJitter(t *testing.T) {
	for attempt:=1;attempt<=20;attempt++{
		delay:=retryDelay(attempt)
		if delay<time.Second||delay>375*time.Second{t.Fatalf("retryDelay(%d) = %s, outside expected bounds",attempt,delay)}
	}
}
