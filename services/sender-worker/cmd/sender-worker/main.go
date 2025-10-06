package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Mutter0815/MassMailer/internal/store"
	"github.com/Mutter0815/MassMailer/pkg/config"
	"github.com/Mutter0815/MassMailer/pkg/db"
	"github.com/Mutter0815/MassMailer/pkg/logx"
	"github.com/Mutter0815/MassMailer/pkg/metrics"
	"github.com/Mutter0815/MassMailer/pkg/rmq"
	"github.com/Mutter0815/MassMailer/services/sender-worker/worker"
)

func main() {

	logx.Init()
	defer logx.Sync()

	config.MustLoadWorker()
	cfg := config.Worker

	sqlDB, err := db.Open(cfg.DBDSN)
	if err != nil {
		logx.L().Fatalw("db_open_error", "error", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			logx.L().Warnf("warn: db close error: %v\n", err)
		}
	}()

	cons, err := rmq.NewConsumer(cfg.RMQURL, cfg.Queue)
	if err != nil {
		logx.L().Fatalw("rmq_consumer_error", "error", err)
	}
	defer func() {
		if err := cons.Close(); err != nil {
			logx.L().Warnf("warn: rmq consumer close error: %v\n", err)
		}
	}()

	pub, err := rmq.NewPublisher(cfg.RMQURL, cfg.Queue)
	if err != nil {
		logx.L().Fatalw("rmq_publisher_error", "error", err)
	}
	defer func() {
		if err := pub.Close(); err != nil {
			logx.L().Warnf("warn: rmq publisher close error: %v\n", err)
		}
	}()

	metricsAddr := getenv("METRICS_ADDR", ":9090")
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())
		logx.L().Infow("metrics_listen", "addr", metricsAddr)
		_ = http.ListenAndServe(metricsAddr, mux)
	}()

	w := worker.New(store.New(sqlDB), cons, pub)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := w.Run(ctx, sqlDB); err != nil && err != context.Canceled {
		logx.L().Fatalw("worker_error", "error", err)
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
