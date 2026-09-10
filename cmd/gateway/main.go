package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
)

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
	mux.Handle("GET /1", proxy)
	mux.Handle("GET /2", proxy)

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
