package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type healthResponse struct {
	Status string `json:"status"`
	Checks struct {
		Postgres string `json:"postgres"`
		Redis    string `json:"redis"`
	} `json:"checks"`
}

func Healthz(dbpool *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := healthResponse{}
		w.Header().Set("Content-Type", "application/json")
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := rdb.Ping(ctx).Err(); err != nil {
			resp.Checks.Redis = "ERROR: Connection unavailable"
		} else {
			resp.Checks.Redis = "OK"
		}

		if err := dbpool.Ping(ctx); err != nil {
			resp.Checks.Postgres = "ERROR: Connection unavailable"
		} else {
			resp.Checks.Postgres = "OK"
		}

		if resp.Checks.Postgres != "OK" || resp.Checks.Redis != "OK" {
			resp.Status = "DOWN"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			resp.Status = "OK"
			w.WriteHeader(http.StatusOK)
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			fmt.Printf("Error encoding json response: %v", err)
		}
	})
}
