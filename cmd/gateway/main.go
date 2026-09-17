package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"api-gateway/internal/middleware"
	"api-gateway/internal/tenants"
	"api-gateway/sql/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var logger = log.New(os.Stderr, "DEBUG: ", log.LstdFlags)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password
		DB:       0,  //default DB
		Protocol: 2,
	})
	defer rdb.Close()
	
	var dbURL string = os.Getenv("DB_URL")
	migrations.RunMigrations(dbURL)
	dbpool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failure connecting to postgres: %v", err)
	}
	defer dbpool.Close()

	repo := tenants.NewRepository(rdb, dbpool)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatalf("Warning: Could not connect to Redis: %v", err)
	} else {
		logger.Println("Successfully connected to Redis!")
	}

	targetURL, err := url.Parse(("http://localhost:8081"))
	if err != nil {
		logger.Fatalf("Invalid target URL: %v", err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(targetURL)
			r.Out.Header.Set("X-Forwarded-Host", r.In.Host)
		},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /1", middleware.RateLimit(repo, rdb, logger, proxy))
	mux.Handle("GET /2", middleware.RateLimit(repo, rdb, logger, proxy))

	const port = "8082"
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	logger.Printf("Gateway serving on port %v\n", port)
	logger.Fatal(srv.ListenAndServe())
}
