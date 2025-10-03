package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewHTTPServer(addr string, h *Handlers) *http.Server {
	r := gin.Default()
	r.GET("/healthz", h.Healthz)
	r.POST("/campaigns", h.CreateCampaign)

	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}
