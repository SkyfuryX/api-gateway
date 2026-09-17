package middleware

import (
	"log"
	"net/http"
	"time"
	"fmt"

	"api-gateway/internal/tenants"

	"github.com/redis/go-redis/v9"
)

func getMinuteKey(apiKey string) string {
	now := time.Now().Unix()
	minuteBucket := now / 60

	return fmt.Sprintf("rate:%s:%d", apiKey, minuteBucket)
}

func RateLimit(repo *tenants.Repository, rdb *redis.Client, logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, "Missing API Key\n", http.StatusUnauthorized)
			return
		}

		tenant, err := repo.GetTenantByAPIKey(r.Context(), apiKey)
		if err != nil {
			logger.Printf("Error retrieving tenant: %v", err)
			http.Error(w, "Invalid API Key", http.StatusForbidden)
			return
		}
		if tenant.Status == "Suspended" {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
		}

		rateKey := getMinuteKey(apiKey)
		count, err := rdb.Incr(r.Context(), rateKey).Result()
		if err != nil {
			log.Printf("Error incrementing key: %v", err)
			http.Error(w, "Internal Server Error\n", http.StatusInternalServerError)
			return
		}
		if count == 1 {
			if err := rdb.Expire(r.Context(), rateKey, 60*time.Second).Err(); err != nil {

			}
		}

		if count > int64(tenant.RateLimitReqPerMin) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
