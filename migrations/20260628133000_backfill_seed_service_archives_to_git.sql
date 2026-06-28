-- +goose Up
WITH matched AS (
    SELECT
        id,
        regexp_match(
            service_archive_url,
            '^https://github\.com/SibirCTF/([^/]+)/archive/refs/heads/([^/]+)\.zip$'
        ) AS captures
    FROM services
)
UPDATE services AS s
SET
    source_kind = 'git',
    git_repo_url = format('https://github.com/SibirCTF/%s.git', matched.captures[1]),
    git_ref = matched.captures[2],
    git_subdir = NULL,
    git_last_commit = NULL,
    git_synced_at = NULL,
    git_sync_status = 'unknown',
    git_sync_error = NULL,
    service_archive_url = NULL,
    checker_archive_url = NULL,
    updated_at = now()
FROM matched
WHERE s.id = matched.id
  AND matched.captures IS NOT NULL
  AND COALESCE(s.git_repo_url, '') = '';

-- +goose Down
WITH matched AS (
    SELECT
        id,
        git_ref,
        regexp_match(
            git_repo_url,
            '^https://github\.com/SibirCTF/([^/]+)\.git$'
        ) AS captures
    FROM services
)
UPDATE services AS s
SET
    service_archive_url = format(
        'https://github.com/SibirCTF/%s/archive/refs/heads/%s.zip',
        matched.captures[1],
        matched.git_ref
    ),
    source_kind = 'manual',
    git_repo_url = NULL,
    git_ref = NULL,
    git_subdir = NULL,
    git_last_commit = NULL,
    git_synced_at = NULL,
    git_sync_status = 'unknown',
    git_sync_error = NULL,
    updated_at = now()
FROM matched
WHERE s.id = matched.id
  AND matched.captures IS NOT NULL
  AND matched.git_ref IS NOT NULL
  AND s.service_archive_url IS NULL;
