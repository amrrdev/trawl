-- ============================================
-- USER AUTHENTICATION QUERIES
-- ============================================

-- name: GetUserByEmail :one
SELECT
    user_id,
    email,
    password,
    name,
    created_at,
    updated_at,
    is_active
FROM users
WHERE email = $1
  AND is_active = true
LIMIT 1;

-- name: GetUserByID :one
SELECT
    user_id,
    email,
    password,
    name,
    created_at,
    updated_at,
    is_active
FROM users
WHERE user_id = $1
  AND is_active = true
LIMIT 1;

-- name: CheckUserExists :one
SELECT COUNT(*) > 0
FROM users
WHERE email = $1;

-- name: GetUserForValidation :one
SELECT
    user_id,
    email,
    is_active
FROM users
WHERE user_id = $1
  AND is_active = true
LIMIT 1;

-- ============================================
-- USER CREATION
-- ============================================

-- name: CreateUser :one
INSERT INTO users (
    email,
    password,
    name
) VALUES (
    $1, $2, $3
)
RETURNING
    user_id,
    email,
    name,
    created_at,
    is_active;

-- ============================================
-- USER PROFILE UPDATES
-- ============================================

-- name: UpdateUserProfile :one
UPDATE users
SET
    name = COALESCE(sqlc.narg('name'), name),
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1
  AND is_active = true
RETURNING
    user_id,
    email,
    name,
    updated_at;

-- name: UpdateUserEmail :one
UPDATE users
SET
    email = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1
  AND is_active = true
RETURNING
    user_id,
    email,
    updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1
  AND is_active = true;

-- ============================================
-- USER STATUS MANAGEMENT
-- ============================================

-- name: DeactivateUser :exec
UPDATE users
SET
    is_active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1;

-- name: ReactivateUser :exec
UPDATE users
SET
    is_active = true,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1;

-- ============================================
-- ADMIN ONLY
-- ============================================

-- name: AdminHardDeleteUser :exec
DELETE FROM users
WHERE user_id = $1;

-- ============================================
-- LISTING & PAGINATION
-- ============================================

-- name: ListUsers :many
SELECT
    user_id,
    email,
    name,
    created_at,
    is_active
FROM users
WHERE
    (sqlc.narg('is_active')::BOOLEAN IS NULL OR is_active = sqlc.narg('is_active'))
    AND (
        sqlc.narg('search') IS NULL OR
        email ILIKE '%' || sqlc.narg('search') || '%' OR
        name ILIKE '%' || sqlc.narg('search') || '%'
    )
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*)
FROM users
WHERE
    (sqlc.narg('is_active')::BOOLEAN IS NULL OR is_active = sqlc.narg('is_active'))
    AND (
        sqlc.narg('search') IS NULL OR
        email ILIKE '%' || sqlc.narg('search') || '%' OR
        name ILIKE '%' || sqlc.narg('search') || '%'
    );

-- name: ListActiveUsers :many
SELECT
    user_id,
    email,
    name,
    created_at
FROM users
WHERE is_active = true
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- ============================================
-- ANALYTICS
-- ============================================

-- name: GetUserStats :one
SELECT
    COUNT(*)                                AS total_users,
    COUNT(*) FILTER (WHERE is_active)       AS active_users,
    COUNT(*) FILTER (WHERE NOT is_active)   AS inactive_users,
    COUNT(*) FILTER (
        WHERE created_at >= NOW() - INTERVAL '24 hours'
    ) AS new_users_24h,
    COUNT(*) FILTER (
        WHERE created_at >= NOW() - INTERVAL '7 days'
    ) AS new_users_7d
FROM users;

-- name: GetUsersByDateRange :many
SELECT
    user_id,
    email,
    name,
    created_at,
    is_active
FROM users
WHERE created_at BETWEEN $1 AND $2
ORDER BY created_at DESC;

-- ============================================
-- BATCH OPERATIONS
-- ============================================

-- name: BulkDeactivateUsers :exec
UPDATE users
SET
    is_active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ANY($1::UUID[]);

-- ============================================
-- DATA INTEGRITY
-- ============================================

-- name: GetDuplicateEmails :many
SELECT
    email,
    COUNT(*) AS count
FROM users
GROUP BY email
HAVING COUNT(*) > 1;
