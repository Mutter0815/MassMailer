package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Mutter0815/MassMailer/docs"
	"github.com/Mutter0815/MassMailer/pkg/metrics"
)

func NewHTTPServer(addr string, h *Handlers) *http.Server {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(Observability())

	r.GET("/healthz", h.Healthz)

	r.GET("/docs", serveSwaggerHTML)
	r.GET("/docs/campaign-api", serveSwaggerHTML)
	r.GET("/docs/campaign-api/openapi.yaml", serveOpenAPI)

	r.POST("/campaigns", h.CreateCampaign)
	r.GET("/campaigns", h.ListCampaigns)
	r.GET("/campaigns/:id", h.GetCampaign)

	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	return &http.Server{Addr: addr, Handler: r}
}

func serveSwaggerHTML(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", docs.CampaignSwaggerHTML)
}

func serveOpenAPI(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.CampaignOpenAPI)
}
