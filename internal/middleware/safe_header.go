package middleware

import "github.com/gin-gonic/gin"

// SafeHeader adds security-related headers to each response
func SafeHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Powered-By", "")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Cache-Control", "no-store")
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}

		c.Next()
	}
}
