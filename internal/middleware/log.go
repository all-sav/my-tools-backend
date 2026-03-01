package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func LoggerMiddleware(log *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if path == "/ws" {
			log.Debug().
				Str("method", method).
				Str("path", path).
				Int("status", status).
				Dur("latency", latency).
				Str("ip", c.ClientIP()).
				Msg("WebSocket connection")
			return
		}

		log.Info().
			Str("method", method).
			Str("path", path).
			Int("status", status).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Msg("Request processed")
	}
}
