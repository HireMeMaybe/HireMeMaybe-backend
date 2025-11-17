package middleware

import (
	"HireMeMaybe-backend/internal/utilities"
	"os"
	"strconv"
	"time"

	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

func keyFunc(c *gin.Context) string {
	user, err := utilities.ExtractUser(c)
	if err != nil {
		return "ip: " + c.ClientIP()
	}
	return "user: " + user.ID.String()
}

func errorHandler(c *gin.Context, info ratelimit.Info) {
	c.AbortWithStatusJSON(429, gin.H{
		"error": "Too many requests. Please try again later.",
	})
}

// RateLimiterMiddleware creates a rate limiter middleware with the specified requests per second.
func RateLimiterMiddleware(reqPerSec uint) gin.HandlerFunc {

	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Second,
		Limit: reqPerSec,
	})

	return ratelimit.RateLimiter(store, &ratelimit.Options{
		KeyFunc:      keyFunc,
		ErrorHandler: errorHandler,
	})
}

// EnvRateLimitMiddleware creates a rate limiter middleware using the RATE_LIMIT_REQUESTS_PER_SECOND environment variable.
func EnvRateLimitMiddleware() gin.HandlerFunc {

	rateLimitString := os.Getenv("RATE_LIMIT_REQUESTS_PER_SECOND")
	var rateLimit uint

	if rateLimitString == "" {
		rateLimit = 5
	} else {
		// Parse as unsigned, clamp to platform max, and ensure non-zero
		if v, err := strconv.ParseUint(rateLimitString, 10, 64); err == nil {
			maxUint := uint64(^uint(0))
			if v == 0 {
				rateLimit = 5
			} else {
				if v > maxUint {
					v = maxUint
				}
				rateLimit = uint(v)
			}
		} else {
			rateLimit = 5
		}
	}

	return RateLimiterMiddleware(rateLimit)
}
