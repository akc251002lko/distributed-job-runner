package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/akc251002lko/distributed-job-runner/internal/config"
	"github.com/akc251002lko/distributed-job-runner/internal/store"
	"github.com/akc251002lko/distributed-job-runner/internal/worker"
)
func main(){logger:=slog.New(slog.NewJSONHandler(os.Stdout,nil));if err:=run(logger);err!=nil{logger.Error("worker stopped","error",err);os.Exit(1)}}
func run(logger *slog.Logger)error{
	cfg,err:=config.Load();if err!=nil{return err}
	ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
	db,err:=store.Open(ctx,cfg.DatabaseURL);if err!=nil{return err};defer db.Close()
	return worker.New(db,cfg.WorkerID,cfg.PollInterval,cfg.LeaseDuration,logger).Run(ctx)
}
