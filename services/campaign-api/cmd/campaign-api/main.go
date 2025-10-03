package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mutter0815/MassMailer/internal/store"
	"github.com/Mutter0815/MassMailer/pkg/config"
	"github.com/Mutter0815/MassMailer/pkg/db"
	"github.com/Mutter0815/MassMailer/pkg/rmq"
	"github.com/Mutter0815/MassMailer/services/campaign-api/server"
)

func main() {
	config.MustLoadAPI()
	cfg := config.API

	sqlDB, err := db.Open(cfg.DBDSN)
	if err != nil {
		log.Fatal("db open:", err)
	}
	defer sqlDB.Close()

	st := store.New(sqlDB)

	pub, err := rmq.NewPublisher(cfg.RMQURL, cfg.Queue)
	if err != nil {
		log.Fatal("rmq:", err)
	}
	defer pub.Close()

	h := server.NewHandlers(st, pub)
	srv := server.NewHTTPServer(":"+cfg.Port, h)

	go func() {
		log.Println("campaign-api listening on :" + cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
