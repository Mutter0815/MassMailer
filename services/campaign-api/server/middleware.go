package server

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Mutter0815/MassMailer/pkg/logx"
	"github.com/Mutter0815/MassMailer/pkg/metrics"
	"github.com/google/uuid"
)

func Observability() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		rid := c.Request.Header.Get("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Writer.Header().Set("X-Request-ID", rid)

		c.Set("request_id", rid)
		c.Next()

		lat := time.Since(start).Seconds()
		status := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		metrics.APIRequestsTotal.WithLabelValues(c.Request.Method, path, strconv.Itoa(status)).Inc()
		metrics.APIRequestDuration.WithLabelValues(c.Request.Method, path).Observe(lat)

		logx.L().Infow("http_access",
			"rid", rid,
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"duration", lat,
			"client_ip", c.ClientIP(),
		)
	}
}
