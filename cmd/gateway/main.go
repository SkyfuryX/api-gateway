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
	"time"

	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"
	"api-gateway/internal/tenants"
	"api-gateway/sql/migrations"

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
		Addr:     "localhost:6379",
		Password: "", // no password
		DB:       0,  //default DB
		Protocol: 2,
	})
	defer rdb.Close()

	var dbURL string = os.Getenv("DB_URL")
	if err := migrations.RunMigrations(logger, dbURL); err != nil {

	}
	dbpool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failure connecting to postgres: %v", err)
	}
	defer dbpool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatalf("Warning: Could not connect to Redis: %v", err)
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
		},
	}

	mux := http.NewServeMux()
	mux.Handle("GET api/1", middleware.RateLimit(repo, rdb, logger, proxy))
	mux.Handle("GET api/2", middleware.RateLimit(repo, rdb, logger, proxy))
	mux.HandleFunc("GET /healthz", handlers.Healthz(dbpool, rdb))

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
