package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
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
