-- name: CreateService :one
INSERT INTO services (name, public_description, private_description, author, copyright,
    avatar_url, public, service_archive_url, checker_archive_url, writeup_url, exploits_url,
    check_status, ctf01d_training, ports, tech_stack, source_kind, git_repo_url, git_ref,
    git_subdir, git_sync_status)
VALUES (
    sqlc.arg('name'),
    sqlc.narg('public_description'),
    sqlc.narg('private_description'),
    sqlc.narg('author'),
    sqlc.narg('copyright'),
    sqlc.narg('avatar_url'),
    sqlc.arg('public'),
    sqlc.narg('service_archive_url'),
    sqlc.narg('checker_archive_url'),
    sqlc.narg('writeup_url'),
    sqlc.narg('exploits_url'),
    sqlc.arg('check_status'),
    sqlc.arg('ctf01d_training'),
    COALESCE(sqlc.arg('ports')::integer[], '{}'),
    COALESCE(sqlc.arg('tech_stack')::text[], '{}'),
    sqlc.arg('source_kind'),
    sqlc.narg('git_repo_url'),
    sqlc.narg('git_ref'),
    sqlc.narg('git_subdir'),
    sqlc.arg('git_sync_status')
)
RETURNING *;

-- name: GetServiceByID :one
SELECT * FROM services WHERE id = $1;

-- name: GetServiceByName :one
SELECT * FROM services WHERE name = $1;

-- name: ListServices :many
SELECT * FROM services
WHERE (public = sqlc.narg('public_filter') OR sqlc.narg('public_filter') IS NULL)
  AND (name ILIKE '%' || sqlc.narg('search_query') || '%' OR sqlc.narg('search_query') IS NULL)
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: CountServices :one
SELECT count(*) FROM services
WHERE (public = sqlc.narg('public_filter') OR sqlc.narg('public_filter') IS NULL)
  AND (name ILIKE '%' || sqlc.narg('search_query') || '%' OR sqlc.narg('search_query') IS NULL);

-- name: UpdateService :one
UPDATE services SET
    name = COALESCE(sqlc.arg('name'), name),
    public_description = COALESCE(sqlc.narg('public_description'), public_description),
    private_description = COALESCE(sqlc.narg('private_description'), private_description),
    author = COALESCE(sqlc.narg('author'), author),
    copyright = COALESCE(sqlc.narg('copyright'), copyright),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    public = COALESCE(sqlc.arg('public'), public),
    service_archive_url = COALESCE(sqlc.narg('service_archive_url'), service_archive_url),
    checker_archive_url = COALESCE(sqlc.narg('checker_archive_url'), checker_archive_url),
    writeup_url = COALESCE(sqlc.narg('writeup_url'), writeup_url),
    exploits_url = COALESCE(sqlc.narg('exploits_url'), exploits_url),
    ctf01d_training = COALESCE(sqlc.arg('ctf01d_training'), ctf01d_training),
    ports = COALESCE(sqlc.arg('ports')::integer[], ports),
    tech_stack = COALESCE(sqlc.arg('tech_stack')::text[], tech_stack),
    updated_at = now()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: SetGitSource :one
UPDATE services SET
    source_kind = $2,
    git_repo_url = $3,
    git_ref = $4,
    git_subdir = $5,
    git_last_commit = NULL,
    git_synced_at = NULL,
    git_sync_status = $6,
    git_sync_error = NULL,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ApplyServiceImportMetadata :one
UPDATE services SET
    name = $2,
    public_description = $3,
    author = $4,
    copyright = $5,
    ctf01d_training = $6,
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

-- name: SetGitSyncState :one
UPDATE services SET
    git_last_commit = $2,
    git_synced_at = $3,
    git_sync_status = $4,
    git_sync_error = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;
