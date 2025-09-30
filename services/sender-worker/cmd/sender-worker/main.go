package main

import (
	"log"
	"time"
)

func main() {
	log.Println("sender-worker started (stub)")
	// временная заглушка — имитируем работу,
	// чтобы контейнер не завершался сразу
	for {
		time.Sleep(10 * time.Second)
	}
}
