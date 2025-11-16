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

func RateLimiterMiddleware(reqPerSec uint) gin.HandlerFunc {

	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:   time.Second,
		Limit:  reqPerSec,
	})

	return ratelimit.RateLimiter(store, &ratelimit.Options{
		KeyFunc:     keyFunc,
		ErrorHandler: errorHandler,
	})
}

func EnvRateLimitMiddleware() gin.HandlerFunc {

	rateLimitString := os.Getenv("RATE_LIMIT_REQUESTS_PER_SECOND")
	rateLimitInt, err := strconv.Atoi(rateLimitString)

	if err != nil {
		rateLimitInt = 5 // default to 5 requests per second if env variable is not set or invalid
	}

	if rateLimitInt <= 0 {
		rateLimitInt = 5 // ensure rate limit is positive
	}

	return RateLimiterMiddleware(uint(rateLimitInt))
}