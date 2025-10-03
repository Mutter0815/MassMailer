package config

import (
	"log"
	"os"
)

type APIConfig struct {
	Port   string
	DBDSN  string
	RMQURL string
	Queue  string
}

type WorkerConfig struct {
	DBDSN  string
	RMQURL string
	Queue  string
}

var (
	API    APIConfig
	Worker WorkerConfig
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("required env %s is not set", k)
	}
	return v
}

func MustLoadAPI() {
	API = APIConfig{
		Port:   getenv("PORT", "8080"),
		DBDSN:  mustEnv("DB_DSN"),
		RMQURL: mustEnv("RMQ_URL"),
		Queue:  getenv("QUEUE", "send_jobs"),
	}
}

func MustLoadWorker() {
	Worker = WorkerConfig{
		DBDSN:  mustEnv("DB_DSN"),
		RMQURL: mustEnv("RMQ_URL"),
		Queue:  getenv("QUEUE", "send_jobs"),
	}
}
