package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/SkyfuryX/api-gateway/internal/handlers"
	"github.com/SkyfuryX/api-gateway/internal/middleware"
	"github.com/SkyfuryX/api-gateway/internal/tenants"
	"github.com/SkyfuryX/api-gateway/sql/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type closeFunc func() error

func initializeLogger() (*log.Logger, closeFunc, error) {
	filename := os.Getenv("LOG_FILE")
	if filename == "" {
		logger := log.New(os.Stderr, "", log.LstdFlags)
		close := func() error { return nil }
		return logger, close, nil
	}
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, nil, fmt.Errorf("Error writing log file: %v", err)

	}
	bufferedFile := bufio.NewWriterSize(f, 8192)
	close := func() error {
		if err = bufferedFile.Flush(); err != nil {
			return fmt.Errorf("Error flushing file buffer: %v", err)
		}
		if err = f.Close(); err != nil {
			return fmt.Errorf("Error closing file: %v", err)
		}
		return nil
	}
	multiWriter := io.MultiWriter(os.Stderr, bufferedFile)
	logger := log.New(multiWriter, "", log.LstdFlags)
	return logger, close, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, relying on system environment variables")
	}

	logger, close, err := initializeLogger()
	defer close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "gateway_redis:6379",
		Password: "", // no password
		DB:       0,  //default DB
		Protocol: 2,
	})
	defer rdb.Close()

	var dbURL string = os.Getenv("DB_URL")
	if err := migrations.RunMigrations(logger, dbURL); err != nil {
		logger.Printf("Error completeing migrations: %v", err)
	}
	dbpool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		logger.Fatalf("Failure creating Postgres pool connection: %v", err)
	}
	defer dbpool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err = dbpool.Ping(context.Background()); err != nil {
		logger.Fatalf("Failure connecting to Postgres: %v", err)
	} else {
		logger.Println("Successfully connected to Postgres!")
	}

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatalf("Failure connecting to Redis: %v", err)
	} else {
		logger.Println("Successfully connected to Redis!")
	}

	repo := tenants.NewRepository(rdb, dbpool)

	targetURL, err := url.Parse(("http://localhost:8081"))
	if err != nil {
		logger.Fatalf("Invalid target URL: %v", err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(targetURL)
			r.Out.Header.Set("X-Forwarded-Host", r.In.Host)
			r.Out.URL.Path = strings.TrimPrefix(r.In.URL.Path, "/api/v1")
		},
	}

	mux := http.NewServeMux()
	mux.Handle("GET api/v1/", middleware.RateLimit(repo, rdb, logger, proxy))
	mux.HandleFunc("GET /healthz", handlers.Healthz(dbpool, rdb))

	const port = "8080"
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	logger.Printf("Gateway serving on port %v\n", port)
	logger.Fatal(srv.ListenAndServe())
}
