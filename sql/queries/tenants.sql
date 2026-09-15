-- name: GetTenantByAPIKey :one
SELECT 
    id,
    name,
    api_key,
    tier,
    rate_limit_req_per_min,
    status,
    created_at,
    updated_at
FROM tenants
WHERE api_key = $1
LIMIT 1;

-- name: CreateTenant :one
INSERT INTO tenants (
    name,
    api_key,
    tier,
    rate_limit_req_per_min,
    status
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, name, api_key, tier, rate_limit_req_per_min, status, created_at;

-- name: UpdateTenantTierAndRate :one
UPDATE tenants
SET 
    rate_limit_req_per_min = $2,
    tier = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, api_key, tier, rate_limit_req_per_min, status, updated_at;

-- name: UpdateTenantStatus :one
UPDATE tenants
    SET status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, status, updated_at;