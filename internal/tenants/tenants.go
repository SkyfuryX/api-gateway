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

func (repo *Repository) GetTenantByAPIKey(ctx context.Context, apiKey string) (db.Tenant, error) {
	cacheKey := fmt.Sprintf("tenant:config:%s", apiKey)

	cachedData, err := repo.Rdb.Get(ctx, cacheKey).Result()
	var tenant db.Tenant
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &tenant); err == nil {
			return tenant, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		fmt.Printf("Redis error on key %s: %v\n", cacheKey, err)
	}

	tenantRow, err := repo.Queries.GetTenantByAPIKey(ctx, apiKey)
	if err != nil {
		return db.Tenant{}, fmt.Errorf("Tenant not found: %w", err)
	}

	serialized, err := json.Marshal(tenantRow)
	if err == nil {
		_ = repo.Rdb.Set(ctx, cacheKey, serialized, tenantCacheTTL)
	}

	tenant = db.Tenant{
		ID:                 tenantRow.ID,
		Name:               tenantRow.Name,
		ApiKey:             tenantRow.ApiKey,
		Tier:               tenantRow.Tier,
		RateLimitReqPerMin: tenantRow.RateLimitReqPerMin,
		Status:             tenantRow.Status,
		CreatedAt:          tenantRow.CreatedAt,
		UpdatedAt:          tenantRow.UpdatedAt,
	}

	return tenant, nil
}

func (repo *Repository) GetTenantByEmail(ctx context.Context, email string) (db.Tenant, error) {
	tenantRow, err := repo.Queries.GetTenantByEmail(context.Background(), email)
	if err != nil {
		return db.Tenant{}, fmt.Errorf("Tenant not found: %w", err)
	}

	tenant := db.Tenant{
		ID:                 tenantRow.ID,
		Name:               tenantRow.Name,
		Email:              tenantRow.Email,
		ApiKey:             tenantRow.ApiKey,
		Tier:               tenantRow.Tier,
		RateLimitReqPerMin: tenantRow.RateLimitReqPerMin,
		Status:             tenantRow.Status,
		CreatedAt:          tenantRow.CreatedAt,
		UpdatedAt:          tenantRow.CreatedAt,
	}
	return tenant, nil
}

func (repo *Repository) CreateTenant(name, email string, tier TenantTier, rate int) (db.Tenant, error) {
	params := db.CreateTenantParams{
		Name:               name,
		Email:              email,
		ApiKey:             GenerateTenantAPIKey(),
		Tier:               string(tier),
		RateLimitReqPerMin: int32(rate),
	}

	tenantRow, err := repo.Queries.CreateTenant(context.Background(), params)
	if err != nil {
		return db.Tenant{}, fmt.Errorf("Error creating new tenant: %w", err)
	}

	tenant := db.Tenant{
		ID:                 tenantRow.ID,
		Name:               tenantRow.Name,
		Email:              tenantRow.Email,
		ApiKey:             tenantRow.ApiKey,
		Tier:               tenantRow.Tier,
		RateLimitReqPerMin: tenantRow.RateLimitReqPerMin,
		Status:             tenantRow.Status,
		UpdatedAt:          tenantRow.CreatedAt,
	}
	return tenant, nil
}

func (repo *Repository) UpdateTierAndRate(id pgtype.UUID, tier TenantTier, rate int) (db.Tenant, error) {
	params := db.UpdateTenantTierAndRateParams{
		ID:                 id,
		Tier:               string(tier),
		RateLimitReqPerMin: int32(rate),
	}

	tenantRow, err := repo.Queries.UpdateTenantTierAndRate(context.Background(), params)
	if err != nil {
		return db.Tenant{}, fmt.Errorf("Error updating tier and rate for ID %v: %w", id, err)
	}

	tenant := db.Tenant{
		ID:                 tenantRow.ID,
		Name:               tenantRow.Name,
		ApiKey:             tenantRow.ApiKey,
		Tier:               tenantRow.Tier,
		RateLimitReqPerMin: tenantRow.RateLimitReqPerMin,
		Status:             tenantRow.Status,
		UpdatedAt:          tenantRow.UpdatedAt,
	}
	return tenant, nil
}

func (repo *Repository) UpdateTenantStatus(id pgtype.UUID, status TenantStatus) (db.Tenant, error) {
	params := db.UpdateTenantStatusParams{
		ID:     id,
		Status: string(status),
	}

	tenantRow, err := repo.Queries.UpdateTenantStatus(context.Background(), params)
	if err != nil {
		return db.Tenant{}, fmt.Errorf("Error updating tenant status for ID %v: %w", id, err)
	}

	tenant := db.Tenant{
		ID:        tenantRow.ID,
		Name:      tenantRow.Name,
		Email:     tenantRow.Email,
		Status:    tenantRow.Status,
		UpdatedAt: tenantRow.UpdatedAt,
	}
	return tenant, nil
}

func (repo *Repository) UpdateTenantAPIKey(id pgtype.UUID) (db.Tenant, error) {
	params := db.UpdateAPIKeyByIDParams{
		ID:     id,
		ApiKey: GenerateTenantAPIKey(),
	}

	tenantRow, err := repo.Queries.UpdateAPIKeyByID(context.Background(), params)
	if err != nil {
		return db.Tenant{}, fmt.Errorf("Error updating tenant APi Key for ID %v: %w", id, err)
	}

	tenant := db.Tenant{
		ID:        tenantRow.ID,
		Name:      tenantRow.Name,
		Email:     tenantRow.Email,
		ApiKey:    tenantRow.ApiKey,
		UpdatedAt: tenantRow.UpdatedAt,
	}
	return tenant, nil
}

func (repo *Repository) InvalidateCache(ctx context.Context, apiKey string) error {
	cacheKey := fmt.Sprintf("tenant:config:%s", apiKey)
	if err := repo.Rdb.Del(ctx, cacheKey).Err(); err != redis.Nil {
		return err
	}
	return nil
}

func GenerateTenantAPIKey() string {
	bytes := make([]byte, 24)
	rand.Read(bytes)
	return fmt.Sprintf("gw_live_%s", hex.EncodeToString(bytes))
}
