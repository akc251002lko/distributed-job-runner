package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL","postgres://localhost/djr")
	t.Setenv("APP_ENV","development")
	t.Setenv("POLL_INTERVAL","")
	t.Setenv("LEASE_DURATION","")
	c,err:=Load();if err!=nil{t.Fatalf("Load(): %v",err)}
	if c.PollInterval!=time.Second||c.LeaseDuration!=30*time.Second{t.Fatalf("unexpected defaults: %+v",c)}
}
func TestProductionRequiresToken(t *testing.T) {
	t.Setenv("DATABASE_URL","postgres://localhost/djr");t.Setenv("APP_ENV","production");t.Setenv("API_TOKEN","")
	if _,err:=Load();err==nil{t.Fatal("expected missing API_TOKEN to fail")}
}
func TestLeaseMustExceedPollingWindow(t *testing.T) {
	t.Setenv("DATABASE_URL","postgres://localhost/djr");t.Setenv("APP_ENV","development")
	t.Setenv("POLL_INTERVAL","10s");t.Setenv("LEASE_DURATION","20s")
	if _,err:=Load();err==nil{t.Fatal("expected short lease to fail")}
}
