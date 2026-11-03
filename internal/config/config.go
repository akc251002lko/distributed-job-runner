package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	AppEnv string
	DatabaseURL string
	APIAddr string
	WorkerMetricsAddr string
	APIToken string
	WorkerID string
	PollInterval time.Duration
	LeaseDuration time.Duration
}

func Load() (Config, error) {
	c := Config{AppEnv: env("APP_ENV", "development"), DatabaseURL: os.Getenv("DATABASE_URL"),
		APIAddr: env("API_ADDR", ":8080"), WorkerMetricsAddr: env("WORKER_METRICS_ADDR", ":9090"), APIToken: os.Getenv("API_TOKEN"), WorkerID: env("WORKER_ID", "worker-local")}
	if c.DatabaseURL == "" { return Config{}, errors.New("DATABASE_URL is required") }
	var err error
	c.PollInterval, err = duration("POLL_INTERVAL", time.Second); if err != nil { return Config{}, err }
	c.LeaseDuration, err = duration("LEASE_DURATION", 30*time.Second); if err != nil { return Config{}, err }
	if c.PollInterval <= 0 { return Config{}, errors.New("POLL_INTERVAL must be positive") }
	if c.LeaseDuration < 3*c.PollInterval { return Config{}, errors.New("LEASE_DURATION must be at least three times POLL_INTERVAL") }
	if c.WorkerID == "" { return Config{}, errors.New("WORKER_ID must not be empty") }
	if c.AppEnv == "production" && c.APIToken == "" { return Config{}, errors.New("API_TOKEN is required in production") }
	return c, nil
}
func env(k, fallback string) string { if v:=os.Getenv(k); v!="" { return v }; return fallback }
func duration(k string, fallback time.Duration) (time.Duration,error) {
	v:=os.Getenv(k); if v=="" { return fallback,nil }; d,err:=time.ParseDuration(v)
	if err!=nil { return 0,fmt.Errorf("%s: %w",k,err) }; return d,nil
}
