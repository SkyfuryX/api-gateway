package tenants

import (
	"api-gateway/internal/db"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	tenantCacheTTL = 5 * time.Minute
)

type Repository struct {
	rdb     *redis.Client
	queries *db.Queries
}

func NewRepository(rdb *redis.Client, dbpool *pgxpool.Pool) *Repository {
	return &Repository{
		rdb:     rdb,
		queries: db.New(dbpool),
	}
}

func (cfg *Repository) GetTenantByAPIKey(ctx context.Context, apiKey string) (*db.Tenant, error) {
	cacheKey := fmt.Sprintf("tenant:config:%s", apiKey)

	cachedData, err := cfg.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var tenant db.Tenant
		if err := json.Unmarshal([]byte(cachedData), &tenant); err == nil {
			return &tenant, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		fmt.Printf("Redis error on key %s: %v\n", cacheKey, err)
	}

	tenant, err := cfg.queries.GetTenantByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch tenant from DB: %w", err)
	}

	serialized, err := json.Marshal(tenant)
	if err == nil {
		_ = cfg.rdb.Set(ctx, cacheKey, serialized, tenantCacheTTL)
	}

	return &tenant, nil
}

func (cfg *Repository) InvalidateCache(ctx context.Context, apiKey string) error {
	cacheKey := fmt.Sprintf("tenant:config:%s", apiKey)
	return cfg.rdb.Del(ctx, cacheKey).Err()
}
