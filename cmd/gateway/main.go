package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
)

func getMinuteKey(apiKey string) string {
	now := time.Now().Unix()
	minuteBucket := now / 60

	return fmt.Sprintf("rate:%s:%d", apiKey, minuteBucket)
}

func rateLimitMiddleware(rdb *redis.Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const limit int64 = 10
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, "Missing API Key\n", http.StatusUnauthorized)
			return
		}

		rateKey := getMinuteKey(apiKey)
		count, err := rdb.Incr(r.Context(), rateKey).Result()
		if err != nil {
			log.Printf("Error incrementing key: %v", err)
			http.Error(w, "Internal Server Error\n", http.StatusInternalServerError)
			return
		}
		if count == 1 {
			rdb.Expire(r.Context(), rateKey, 60*time.Second)
		}

		if count > limit {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password
		DB:       0,  //default DB
		Protocol: 2,
	})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Warning: Could not connect to Redis: %v", err)
	} else {
		log.Println("Successfully connected to Redis!")
	}

	targetURL, err := url.Parse(("http://localhost:8081"))
	if err != nil {
		log.Fatalf("Invalid target URL: %v", err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(targetURL)
			r.Out.Header.Set("X-Forwarded-Host", r.In.Host)
		},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /1", rateLimitMiddleware(rdb, proxy))
	mux.Handle("GET /2", rateLimitMiddleware(rdb, proxy))

	const port = "8080"
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Printf("Gateway serving on port %v\n", port)
	log.Fatal(srv.ListenAndServe())
}
