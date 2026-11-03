package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akc251002lko/distributed-job-runner/internal/config"
	"github.com/akc251002lko/distributed-job-runner/internal/observability"
	"github.com/akc251002lko/distributed-job-runner/internal/store"
	"github.com/akc251002lko/distributed-job-runner/internal/worker"
)

func main() {
	logger:=slog.New(slog.NewJSONHandler(os.Stdout,nil))
	if err:=run(logger);err!=nil{logger.Error("worker stopped","error",err);os.Exit(1)}
}

func run(logger *slog.Logger)error {
	cfg,err:=config.Load();if err!=nil{return err}
	ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
	db,err:=store.Open(ctx,cfg.DatabaseURL);if err!=nil{return err};defer db.Close()

	metricsServer:=&http.Server{Addr:cfg.WorkerMetricsAddr,Handler:observability.Handler(),ReadHeaderTimeout:5*time.Second}
	metricsErr:=make(chan error,1)
	go func(){
		if serveErr:=metricsServer.ListenAndServe();serveErr!=nil&&!errors.Is(serveErr,http.ErrServerClosed){metricsErr<-serveErr}
	}()
	runnerErr:=make(chan error,1)
	go func(){runnerErr<-worker.New(db,cfg.WorkerID,cfg.PollInterval,cfg.LeaseDuration,logger).Run(ctx)}()

	var runnerResult error
	var metricsFailure error
	runnerFinished:=false
	select {
	case <-ctx.Done():
	case runnerResult=<-runnerErr:
		runnerFinished=true
		stop()
	case serveErr:=<-metricsErr:
		logger.Error("worker metrics server failed","error",serveErr)
		metricsFailure=serveErr
		stop()
	}
	shutdownCtx,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel()
	if err:=metricsServer.Shutdown(shutdownCtx);err!=nil{return err}
	if !runnerFinished {
		select {
		case runnerResult=<-runnerErr:
		case <-shutdownCtx.Done():return shutdownCtx.Err()
		}
	}
	if metricsFailure!=nil{return metricsFailure}
	return runnerResult
}
