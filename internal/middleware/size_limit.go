package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var multipartOverhead = int64(8 * 1024) // rough padding

// SizeLimit function is a middleware that check if file is larger than maxBodyBytes or not
// will return http.MaxBytesError when file size exceed maxBodyBytes
// and usually response with 413 request entity too large.
func SizeLimit(maxBodyBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		w := c.Writer

		c.Request.Body = http.MaxBytesReader(w, c.Request.Body, maxBodyBytes+(c.Request.ContentLength+multipartOverhead))

		c.Next()
	}
}
