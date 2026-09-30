-- name: CreateAdminUser :one
INSERT INTO admin_users (
    name,
    email,
    api_key
) VALUES (
    $1, $2, $3
)
RETURNING id, name, email, api_key, status, created_at;

-- name: GetAdminByAPIKey :one
SELECT 
    id,
    name,
    api_key,
    status
FROM admin_users
WHERE api_key = $1
LIMIT 1;

-- name: UpdateAdminStatus :one
UPDATE admin_users
    SET status = $2,
    updated_at = NOW()
WHERE email = $1
RETURNING id, name, email, status, updated_at;

-- name: UpdateAdminAPIKey :one
UPDATE admin_users
    SET
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, status, updated_at;