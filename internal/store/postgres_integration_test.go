package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
	"sync"
)

func integrationStore(t *testing.T)(*Store,context.Context){
	t.Helper()
	dsn:=os.Getenv("DATABASE_URL")
	if dsn==""{t.Skip("DATABASE_URL is not set")}
	ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second)
	t.Cleanup(cancel)
	db,err:=Open(ctx,dsn);if err!=nil{t.Fatalf("Open(): %v",err)}
	t.Cleanup(db.Close)
	return db,ctx
}

func TestEnqueueIdempotencyLeaseRenewalAndRecovery(t *testing.T) {
	db,ctx:=integrationStore(t)
	key:=fmt.Sprintf("integration-%d",time.Now().UnixNano())
	payload:=json.RawMessage(`{"test":true}`)
	first,err:=db.Enqueue(ctx,"noop",payload,key,3,time.Now().UTC());if err!=nil{t.Fatalf("Enqueue(): %v",err)}
	second,err:=db.Enqueue(ctx,"noop",payload,key,3,time.Now().UTC());if err!=nil{t.Fatalf("idempotent Enqueue(): %v",err)}
	if first.ID!=second.ID{t.Fatalf("idempotency returned different IDs: %s != %s",first.ID,second.ID)}
	if _,err:=db.Enqueue(ctx,"noop",json.RawMessage(`{"test":"different"}`),key,3,time.Now().UTC());!errors.Is(err,ErrIdempotencyConflict){
		t.Fatalf("expected ErrIdempotencyConflict for changed payload, got %v",err)
	}

	claimed,err:=db.Claim(ctx,"integration-worker-1",30*time.Second);if err!=nil{t.Fatalf("Claim(): %v",err)}
	if claimed.ID!=first.ID{t.Fatalf("claimed ID = %s, want %s",claimed.ID,first.ID)}
	if err:=db.RenewLease(ctx,claimed.ID,claimed.LeaseToken,30*time.Second);err!=nil{t.Fatalf("RenewLease(): %v",err)}
	if err:=db.Complete(ctx,claimed.ID,"stale-token");err==nil{t.Fatal("stale lease token unexpectedly completed job")}

	if _,err:=db.pool.Exec(ctx,`UPDATE jobs SET lease_expires_at=now()-interval '1 second' WHERE id=$1`,claimed.ID);err!=nil{t.Fatalf("expire lease: %v",err)}
	if err:=db.Complete(ctx,claimed.ID,claimed.LeaseToken);err==nil{t.Fatal("expired lease unexpectedly completed job")}

	reclaimed,err:=db.Claim(ctx,"integration-worker-2",30*time.Second);if err!=nil{t.Fatalf("reclaim expired job: %v",err)}
	if reclaimed.ID!=first.ID{t.Fatalf("reclaimed ID = %s, want %s",reclaimed.ID,first.ID)}
	if reclaimed.LeaseToken==claimed.LeaseToken{t.Fatal("reclaimed job reused old lease token")}
	if err:=db.Complete(ctx,reclaimed.ID,reclaimed.LeaseToken);err!=nil{t.Fatalf("Complete(): %v",err)}
	final,err:=db.Get(ctx,first.ID);if err!=nil{t.Fatalf("Get(): %v",err)}
	if final.Status!="succeeded"{t.Fatalf("status = %s, want succeeded",final.Status)}

	deadJob,err:=db.Enqueue(ctx,"noop",payload,fmt.Sprintf("exhausted-%d",time.Now().UnixNano()),1,time.Now().UTC())
	if err!=nil{t.Fatalf("enqueue final-attempt job: %v",err)}
	lastAttempt,err:=db.Claim(ctx,"integration-worker-3",30*time.Second)
	if err!=nil{t.Fatalf("claim final-attempt job: %v",err)}
	if lastAttempt.ID!=deadJob.ID{t.Fatalf("final-attempt claim ID = %s, want %s",lastAttempt.ID,deadJob.ID)}
	if _,err:=db.pool.Exec(ctx,`UPDATE jobs SET lease_expires_at=now()-interval '1 second' WHERE id=$1`,deadJob.ID);err!=nil{t.Fatalf("expire final-attempt lease: %v",err)}
	afterExpiry,err:=db.Claim(ctx,"integration-worker-4",30*time.Second)
	if err!=nil{t.Fatalf("claim after exhausted lease: %v",err)}
	if afterExpiry.ID!=""{t.Fatalf("exhausted job was reclaimed: %s",afterExpiry.ID)}
	deadState,err:=db.Get(ctx,deadJob.ID);if err!=nil{t.Fatalf("read exhausted job: %v",err)}
	if deadState.Status!="dead"{t.Fatalf("exhausted job status = %s, want dead",deadState.Status)}
}

func TestConcurrentClaimsAreUnique(t *testing.T) {
	db,ctx:=integrationStore(t)
	const count=8
	for i:=0;i<count;i++ {
		key:=fmt.Sprintf("concurrent-claim-%d-%d",time.Now().UnixNano(),i)
		if _,err:=db.Enqueue(ctx,"noop",json.RawMessage(`{"concurrent":true}`),key,3,time.Now().UTC());err!=nil {
			t.Fatalf("enqueue job %d: %v",i,err)
		}
	}
	type claimResult struct { id string; token string; err error }
	results:=make(chan claimResult,count)
	var wg sync.WaitGroup
	for i:=0;i<count;i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			claimCtx,cancel:=context.WithTimeout(ctx,10*time.Second)
			defer cancel()
			j,err:=db.Claim(claimCtx,fmt.Sprintf("concurrent-worker-%d",worker),30*time.Second)
			results<-claimResult{id:j.ID,token:j.LeaseToken,err:err}
		}(i)
	}
	wg.Wait()
	close(results)
	seen:=make(map[string]struct{},count)
	for result:=range results {
		if result.err!=nil { t.Fatalf("concurrent claim: %v",result.err) }
		if result.id=="" { t.Fatal("concurrent claim returned no job") }
		if _,exists:=seen[result.id];exists { t.Fatalf("job %s was claimed more than once",result.id) }
		seen[result.id]=struct{}{}
		if err:=db.Complete(ctx,result.id,result.token);err!=nil { t.Fatalf("complete claimed job %s: %v",result.id,err) }
	}
	if len(seen)!=count { t.Fatalf("unique claims = %d, want %d",len(seen),count) }
}


func TestListCancelAndReplayLifecycle(t *testing.T) {
	db,ctx:=integrationStore(t)
	key1:=fmt.Sprintf("lifecycle-a-%d",time.Now().UnixNano())
	key2:=fmt.Sprintf("lifecycle-b-%d",time.Now().UnixNano())
	first,err:=db.Enqueue(ctx,"noop",json.RawMessage("{\"lifecycle\":1}"),key1,3,time.Now().UTC())
	if err!=nil{t.Fatalf("enqueue first: %v",err)}
	second,err:=db.Enqueue(ctx,"noop",json.RawMessage("{\"lifecycle\":2}"),key2,3,time.Now().UTC())
	if err!=nil{t.Fatalf("enqueue second: %v",err)}
	if _,err=db.Cancel(ctx,first.ID);err!=nil{t.Fatalf("cancel: %v",err)}
	if _,err=db.Cancel(ctx,second.ID);err!=nil{t.Fatalf("cancel second: %v",err)}
	page,err:=db.List(ctx,1,"cancelled","")
	if err!=nil{t.Fatalf("list: %v",err)}
	if len(page.Items)!=1||page.NextCursor==""{t.Fatalf("first page = %+v, expected one item and next cursor",page)}
	next,err:=db.List(ctx,1,"cancelled",page.NextCursor)
	if err!=nil{t.Fatalf("list next page: %v",err)}
	if len(next.Items)!=1||next.Items[0].ID==page.Items[0].ID{t.Fatalf("second page = %+v",next)}
	replayed,err:=db.Replay(ctx,first.ID)
	if err!=nil{t.Fatalf("replay: %v",err)}
	if replayed.ID==first.ID||replayed.Status!="pending"||replayed.Attempts!=0{t.Fatalf("unexpected replay: %+v",replayed)}
	if _,err=db.Replay(ctx,replayed.ID);!errors.Is(err,ErrInvalidTransition){t.Fatalf("replay of pending job error = %v, want ErrInvalidTransition",err)}
	if _,err=db.Cancel(ctx,replayed.ID);err!=nil{t.Fatalf("cleanup replayed job: %v",err)}
}


func TestClientDataIsolationAndScopedIdempotency(t *testing.T) {
	db,ctx:=integrationStore(t)
	key:=fmt.Sprintf("tenant-key-%d",time.Now().UnixNano())
	a,err:=db.EnqueueForClient(ctx,"tenant-a","noop",json.RawMessage("{}"),key,2,time.Now().UTC())
	if err!=nil{t.Fatalf("enqueue tenant a: %v",err)}
	b,err:=db.EnqueueForClient(ctx,"tenant-b","noop",json.RawMessage("{}"),key,2,time.Now().UTC())
	if err!=nil{t.Fatalf("same key for tenant b should be independent: %v",err)}
	if a.ID==b.ID{t.Fatal("different clients received the same job")}
	if _,err:=db.GetForClient(ctx,"tenant-b",a.ID);!errors.Is(err,ErrNotFound){t.Fatalf("cross-client get error = %v",err)}
	page,err:=db.ListForClient(ctx,"tenant-b",100,"","")
	if err!=nil{t.Fatalf("tenant b list: %v",err)}
	for _,item:=range page.Items{if item.ID==a.ID{t.Fatal("tenant b list exposed tenant a job")}}
	if _,err:=db.CancelForClient(ctx,"tenant-b",a.ID);!errors.Is(err,ErrNotFound){t.Fatalf("cross-client cancel error = %v",err)}
	if _,err:=db.ReplayForClient(ctx,"tenant-b",a.ID);!errors.Is(err,ErrNotFound){t.Fatalf("cross-client replay error = %v",err)}
}


func TestWorkerDatabaseRoleCanClaimAndCompleteAcrossClients(t *testing.T) {
	apiDSN:=os.Getenv("API_DATABASE_URL")
	workerDSN:=os.Getenv("WORKER_DATABASE_URL")
	if apiDSN==""||workerDSN==""{t.Skip("API_DATABASE_URL and WORKER_DATABASE_URL are required")}
	ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second)
	t.Cleanup(cancel)
	apiDB,err:=Open(ctx,apiDSN)
	if err!=nil{t.Fatalf("open API role: %v",err)}
	t.Cleanup(apiDB.Close)
	workerDB,err:=Open(ctx,workerDSN)
	if err!=nil{t.Fatalf("open worker role: %v",err)}
	t.Cleanup(workerDB.Close)
	job,err:=apiDB.EnqueueForClient(ctx,"worker-tenant","noop",json.RawMessage("{}"),fmt.Sprintf("worker-role-%d",time.Now().UnixNano()),1,time.Now().UTC())
	if err!=nil{t.Fatalf("enqueue through API role: %v",err)}
	var targetClaimed bool
	for attempt:=0;attempt<100;attempt++ {
		claimed,claimErr:=workerDB.Claim(ctx,"restricted-worker",30*time.Second)
		if claimErr!=nil{t.Fatalf("claim through worker role: %v",claimErr)}
		if claimed.ID==""{break}
		if err:=workerDB.Complete(ctx,claimed.ID,claimed.LeaseToken);err!=nil{t.Fatalf("complete through worker role: %v",err)}
		if claimed.ID==job.ID{targetClaimed=true;break}
	}
	if !targetClaimed{t.Fatalf("worker role did not claim submitted job %s",job.ID)}
}
