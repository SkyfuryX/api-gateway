package tenants

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/SkyfuryX/api-gateway/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	tenantCacheTTL = 5 * time.Minute

	StatusActive    TenantStatus = "active"
	StatusSuspended TenantStatus = "suspended"

	TierFree       TenantTier = "free"
	TierPro        TenantTier = "pro"
	TierEnterprise TenantTier = "enterprise"
)

type TenantStatus string
type TenantTier string
type Repository struct {
	Rdb     *redis.Client
	Queries *db.Queries
}

func NewRepository(rdb *redis.Client, dbpool *pgxpool.Pool) *Repository {
	return &Repository{
		Rdb:     rdb,
		Queries: db.New(dbpool),
	}
}

func (repo *Repository) GetTenantByAPIKey(ctx context.Context, apiKey string) (*db.Tenant, error) {
	cacheKey := fmt.Sprintf("tenant:config:%s", apiKey)

	cachedData, err := repo.Rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var tenant db.Tenant
		if err := json.Unmarshal([]byte(cachedData), &tenant); err == nil {
			return &tenant, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		fmt.Printf("Redis error on key %s: %v\n", cacheKey, err)
	}

	tenant, err := repo.Queries.GetTenantByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch tenant from DB: %w", err)
	}

	serialized, err := json.Marshal(tenant)
	if err == nil {
		_ = repo.Rdb.Set(ctx, cacheKey, serialized, tenantCacheTTL)
	}

	return &tenant, nil
}

func (repo *Repository) CreateTenant(name string, tier TenantTier, rate int, status TenantStatus) (db.CreateTenantRow, error) {
	params := db.CreateTenantParams{
		Name:               name,
		ApiKey:             GenerateAPIKey(),
		Tier:               string(tier),
		RateLimitReqPerMin: int32(rate),
		Status:             string(status),
	}

	tenant, err := repo.Queries.CreateTenant(context.Background(), params)
	if err != nil {
		return db.CreateTenantRow{}, fmt.Errorf("Error creating new tenant: %v", err)
	}

	return tenant, nil
}

func (repo *Repository) UpdateTierAndRate(id pgtype.UUID, tier TenantTier, rate int) (db.UpdateTenantTierAndRateRow, error) {
	params := db.UpdateTenantTierAndRateParams{
		ID:                 id,
		Tier:               string(tier),
		RateLimitReqPerMin: int32(rate),
	}

	tenant, err := repo.Queries.UpdateTenantTierAndRate(context.Background(), params)
	if err != nil {
		return db.UpdateTenantTierAndRateRow{}, fmt.Errorf("Error updating tier and rate for ID %v: %v", id, err)
	}

	return tenant, nil
}

func (repo *Repository) UpdateTenantStatus(id pgtype.UUID, status TenantStatus) (db.UpdateTenantStatusRow, error) {
	params := db.UpdateTenantStatusParams{
		ID:     id,
		Status: string(status),
	}

	statusRow, err := repo.Queries.UpdateTenantStatus(context.Background(), params)
	if err != nil {
		return db.UpdateTenantStatusRow{}, fmt.Errorf("Error updating tenant status for ID %v: %v", id, err)
	}
	return statusRow, nil
}

func (repo *Repository) InvalidateCache(ctx context.Context, apiKey string) error {
	cacheKey := fmt.Sprintf("tenant:config:%s", apiKey)
	return repo.Rdb.Del(ctx, cacheKey).Err()
}

func GenerateAPIKey() string {
	bytes := make([]byte, 24)
	rand.Read(bytes)
	return fmt.Sprintf("gw_live_%s", hex.EncodeToString(bytes))
}
