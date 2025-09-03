package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SizeLimit(maxBodyBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		w := c.Writer
		c.Request.Body = http.MaxBytesReader(w, c.Request.Body, maxBodyBytes)

		c.Next()
	}
}
