package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestEnqueueIdempotencyAndClaimLifecycle(t *testing.T) {
	dsn:=os.Getenv("DATABASE_URL");if dsn==""{t.Skip("DATABASE_URL is not set")}
	ctx,cancel:=context.WithTimeout(context.Background(),15*time.Second);defer cancel()
	db,err:=Open(ctx,dsn);if err!=nil{t.Fatalf("Open(): %v",err)};defer db.Close()
	key:=fmt.Sprintf("integration-%d",time.Now().UnixNano())
	payload:=json.RawMessage(`{"test":true}`)
	first,err:=db.Enqueue(ctx,"noop",payload,key,3,time.Now().UTC());if err!=nil{t.Fatalf("Enqueue(): %v",err)}
	second,err:=db.Enqueue(ctx,"noop",payload,key,3,time.Now().UTC());if err!=nil{t.Fatalf("idempotent Enqueue(): %v",err)}
	if first.ID!=second.ID{t.Fatalf("idempotency returned different IDs: %s != %s",first.ID,second.ID)}
	claimed,err:=db.Claim(ctx,"integration-worker",30*time.Second);if err!=nil{t.Fatalf("Claim(): %v",err)}
	if claimed.ID!=first.ID{t.Fatalf("claimed ID = %s, want %s",claimed.ID,first.ID)}
	if err:=db.Complete(ctx,claimed.ID,claimed.LeaseToken);err!=nil{t.Fatalf("Complete(): %v",err)}
	final,err:=db.Get(ctx,first.ID);if err!=nil{t.Fatalf("Get(): %v",err)}
	if final.Status!="succeeded"{t.Fatalf("status = %s, want succeeded",final.Status)}
}
