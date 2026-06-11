-- name: CreateService :one
INSERT INTO services (name, public_description, private_description, author, copyright,
    avatar_url, public, service_archive_url, checker_archive_url, writeup_url, exploits_url,
    check_status, ctf01d_training)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetServiceByID :one
SELECT * FROM services WHERE id = $1;

-- name: GetServiceByName :one
SELECT * FROM services WHERE name = $1;

-- name: ListServices :many
SELECT * FROM services
WHERE ($1::boolean IS NULL OR public = $1)
  AND ($2::text IS NULL OR name ILIKE '%' || $2 || '%')
ORDER BY id
LIMIT $3 OFFSET $4;

-- name: CountServices :one
SELECT count(*) FROM services
WHERE ($1::boolean IS NULL OR public = $1)
  AND ($2::text IS NULL OR name ILIKE '%' || $2 || '%');

-- name: UpdateService :one
UPDATE services SET
    name = COALESCE($2, name),
    public_description = COALESCE($3, public_description),
    private_description = COALESCE($4, private_description),
    author = COALESCE($5, author),
    copyright = COALESCE($6, copyright),
    avatar_url = COALESCE($7, avatar_url),
    public = COALESCE($8, public),
    service_archive_url = COALESCE($9, service_archive_url),
    checker_archive_url = COALESCE($10, checker_archive_url),
    writeup_url = COALESCE($11, writeup_url),
    exploits_url = COALESCE($12, exploits_url),
    ctf01d_training = COALESCE($13, ctf01d_training),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteService :exec
DELETE FROM services WHERE id = $1;

-- name: SetPublic :one
UPDATE services SET public = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetCheckStatus :one
UPDATE services SET check_status = $2, checked_at = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetServiceLocal :one
UPDATE services SET
    service_local_path = $2,
    service_local_size = $3,
    service_local_sha256 = $4,
    service_downloaded_at = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetCheckerLocal :one
UPDATE services SET
    checker_local_path = $2,
    checker_local_size = $3,
    checker_local_sha256 = $4,
    checker_downloaded_at = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetArchiveURLs :one
UPDATE services SET
    service_archive_url = COALESCE($2, service_archive_url),
    checker_archive_url = COALESCE($3, checker_archive_url),
    updated_at = now()
WHERE id = $1
RETURNING *;
