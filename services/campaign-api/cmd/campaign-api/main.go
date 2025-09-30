package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	port := getenv("PORT", "8080")

	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) { c.String(200, "ok") })

	log.Println("campaign-api listening on :" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
