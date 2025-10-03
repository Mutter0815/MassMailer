package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/Mutter0815/MassMailer/internal/store"
	"github.com/Mutter0815/MassMailer/pkg/config"
	"github.com/Mutter0815/MassMailer/pkg/db"
	"github.com/Mutter0815/MassMailer/pkg/rmq"
	"github.com/Mutter0815/MassMailer/services/sender-worker/worker"
)

func main() {
	config.MustLoadWorker()
	cfg := config.Worker

	sqlDB, err := db.Open(cfg.DBDSN)
	if err != nil {
		log.Fatal("db open:", err)
	}
	defer sqlDB.Close()

	cons, err := rmq.NewConsumer(cfg.RMQURL, cfg.Queue)
	if err != nil {
		log.Fatal("rmq:", err)
	}
	defer cons.Close()

	w := worker.New(store.New(sqlDB), cons)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := w.Run(ctx, sqlDB); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
