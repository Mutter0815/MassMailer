package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mutter0815/MassMailer/internal/store"
	"github.com/Mutter0815/MassMailer/pkg/config"
	"github.com/Mutter0815/MassMailer/pkg/db"
	"github.com/Mutter0815/MassMailer/pkg/logx"
	"github.com/Mutter0815/MassMailer/pkg/rmq"
	"github.com/Mutter0815/MassMailer/services/campaign-api/server"
)

func main() {
	logx.Init()
	defer logx.Sync()

	config.MustLoadAPI()
	cfg := config.API

	sqlDB, err := db.Open(cfg.DBDSN)
	if err != nil {
		logx.L().Fatalw("db_open_error", "error", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			logx.L().Warnw("db_close_error", "error", err)
		} else {
			logx.L().Infow("db_closed")
		}
	}()

	st := store.New(sqlDB)

	pub, err := rmq.NewPublisher(cfg.RMQURL, cfg.Queue)
	if err != nil {
		logx.L().Fatalw("rmq_init_error", "error", err)
	}
	defer func() {
		if err := pub.Close(); err != nil {
			logx.L().Warnw("rmq_publisher_close_error", "error", err)
		} else {
			logx.L().Infow("rmq_publisher_closed")
		}
	}()

	h := server.NewHandlers(st, pub)
	srv := server.NewHTTPServer(":"+cfg.Port, h)

	go func() {
		logx.L().Infow("api_listen_start", "addr", ":"+cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logx.L().Fatalw("http_server_error", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	sig := <-stop
	logx.L().Infow("signal_received", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logx.L().Errorw("server_shutdown_error", "error", err)
	} else {
		logx.L().Infow("server_shutdown_success")
	}

	logx.L().Infow("campaign-api stopped gracefully")
}
