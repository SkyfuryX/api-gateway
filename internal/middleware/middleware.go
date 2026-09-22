package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

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
			return
		}

		rateKey := getMinuteKey(apiKey)

		var incrCmd *redis.IntCmd
		for range 3 {
			pipe := rdb.Pipeline()
			incrCmd = pipe.Incr(r.Context(), rateKey)
			pipe.ExpireNX(r.Context(), rateKey, 60*time.Second)

			_, err = pipe.Exec(r.Context())
			if err == nil {
				break // Pipeline executed successfully
			}

			time.Sleep(10 * time.Millisecond) // Short pause before retrying
		}

		if err != nil {
			logger.Printf("Failed to execute rate limit pipeline on %s: %v. Deleting key.", rateKey, err)
			_ = rdb.Del(r.Context(), rateKey).Err() // Delete orphan key
			http.Error(w, "Internal Server Error\n", http.StatusInternalServerError)
			return
		}

		count := incrCmd.Val()

		if count > int64(tenant.RateLimitReqPerMin) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
