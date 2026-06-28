package services

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	localRepoHost  = "local"
	defaultRepoDir = "repo"
)

type GitImportRequest struct {
	RepoURL string
	Ref     string
	Subdir  string
}

type GitSourceInput struct {
	RepoURL string
	Ref     string
	Subdir  string
}

type gitRepoReference struct {
	CloneURL string
	RepoURL  string
	Host     string
	RepoPath string
	Owner    string
	Repo     string
	Ref      string
	Subdir   string
}

type fetchedGitRepo struct {
	ZipBytes []byte
	Commit   string
	Ref      string
	RepoURL  string
	Subdir   string
	Source   importSourceInfo
}

type gitArchiveFetcher interface {
	Fetch(ctx context.Context, req GitImportRequest) (*fetchedGitRepo, error)
}

type execGitArchiveFetcher struct {
	maxArchiveBytes int64
}

func newExecGitArchiveFetcher(maxArchiveBytes int64) *execGitArchiveFetcher {
	return &execGitArchiveFetcher{maxArchiveBytes: maxArchiveBytes}
}

func (f *execGitArchiveFetcher) Fetch(ctx context.Context, req GitImportRequest) (*fetchedGitRepo, error) {
	ref, err := normalizeGitImportRequest(req)
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "ctf01d-git-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneDir := filepath.Join(tmpDir, defaultRepoDir)
	if _, err := runGitCommand(ctx, "", "clone", "--depth", "1", "--single-branch", ref.CloneURL, cloneDir); err != nil {
		return nil, fmt.Errorf("git clone: %w", err)
	}
	if err := ensureDirectorySizeLimit(cloneDir, effectiveGitSizeLimit(f.maxArchiveBytes)); err != nil {
		return nil, err
	}

	requestedRef := ref.Ref
	if requestedRef != "" {
		if _, err := runGitCommand(ctx, cloneDir, "fetch", "--depth", "1", "origin", requestedRef); err != nil {
			return nil, fmt.Errorf("git fetch %q: %w", requestedRef, err)
		}
		if _, err := runGitCommand(ctx, cloneDir, "checkout", "--detach", "FETCH_HEAD"); err != nil {
			return nil, fmt.Errorf("git checkout %q: %w", requestedRef, err)
		}
	}
	if err := ensureDirectorySizeLimit(cloneDir, effectiveGitSizeLimit(f.maxArchiveBytes)); err != nil {
		return nil, err
	}

	commit, err := runGitCommand(ctx, cloneDir, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("git rev-parse: %w", err)
	}

	selectedDir := cloneDir
	rootName := ref.Repo
	if ref.Subdir != "" {
		selectedDir = filepath.Join(cloneDir, filepath.FromSlash(ref.Subdir))
		rootName = path.Base(ref.Subdir)
	}
	if rootName == "" {
		rootName = defaultRepoDir
	}

	zipBytes, err := zipDirectory(selectedDir, rootName, f.maxArchiveBytes)
	if err != nil {
		return nil, err
	}
	if err := validateZipBytes(zipBytes); err != nil {
		return nil, err
	}

	return &fetchedGitRepo{
		ZipBytes: zipBytes,
		Commit:   strings.TrimSpace(string(commit)),
		Ref:      requestedRef,
		RepoURL:  ref.RepoURL,
		Subdir:   ref.Subdir,
		Source: importSourceInfo{
			Source: sourceGit,
			Owner:  ref.Owner,
			Repo:   ref.Repo,
			Host:   ref.Host,
			Path:   ref.RepoPath,
		},
	}, nil
}

func normalizeGitSourceInput(input *GitSourceInput) (string, *string, *string, *string, error) {
	if input == nil {
		return sourceManual, nil, nil, nil, nil
	}

	repoURL := strings.TrimSpace(input.RepoURL)
	if repoURL == "" {
		return sourceManual, nil, nil, nil, nil
	}

	ref, err := normalizeGitImportRequest(GitImportRequest{
		RepoURL: repoURL,
		Ref:     input.Ref,
		Subdir:  input.Subdir,
	})
	if err != nil {
		return "", nil, nil, nil, err
	}

	repoURLPtr := &ref.RepoURL
	refPtr := optionalTrimmedString(ref.Ref)
	subdirPtr := optionalTrimmedString(ref.Subdir)
	return sourceGit, repoURLPtr, refPtr, subdirPtr, nil
}

func normalizeGitImportRequest(req GitImportRequest) (*gitRepoReference, error) {
	repoURL := strings.TrimSpace(req.RepoURL)
	if repoURL == "" {
		return nil, errors.New("repo_url is required")
	}

	ref, err := parseGitRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.Ref) != "" {
		ref.Ref = strings.TrimSpace(req.Ref)
	}

	subdir := strings.TrimSpace(req.Subdir)
	if subdir != "" {
		clean := safeRelPath(subdir)
		if clean == "" {
			return nil, errors.New("subdir must be a relative path without traversal")
		}
		ref.Subdir = clean
	}

	return ref, nil
}

func parseGitRepoURL(repoURL string) (*gitRepoReference, error) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return nil, errors.New("repo_url is required")
	}

	if looksLikeLocalGitPath(repoURL) {
		repo := strings.TrimSuffix(filepath.Base(repoURL), ".git")
		return &gitRepoReference{
			CloneURL: repoURL,
			RepoURL:  repoURL,
			Host:     localRepoHost,
			RepoPath: filepath.Clean(repoURL),
			Repo:     repo,
		}, nil
	}

	if !strings.Contains(repoURL, "://") && strings.Contains(repoURL, ":") {
		at := strings.Index(repoURL, "@")
		colon := strings.Index(repoURL, ":")
		if colon > at {
			hostPart := repoURL[:colon]
			pathPart := strings.TrimPrefix(repoURL[colon+1:], "/")
			host := hostPart
			if at >= 0 {
				host = hostPart[at+1:]
			}
			return buildGitRepoReference(repoURL, repoURL, host, pathPart, "")
		}
	}

	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid git URL: %w", err)
	}
	if parsedURL.Scheme == "" {
		return nil, errors.New("repo_url must be a valid git URL or an existing local path")
	}
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "ssh" && parsedURL.Scheme != "git" && parsedURL.Scheme != "file" {
		return nil, fmt.Errorf("unsupported git URL scheme %q", parsedURL.Scheme)
	}

	ref := strings.TrimSpace(parsedURL.Fragment)
	parsedURL.Fragment = ""
	cloneURL := parsedURL.String()
	host := parsedURL.Hostname()
	repoPath := strings.TrimPrefix(parsedURL.Path, "/")
	if parsedURL.Scheme == "file" {
		host = localRepoHost
		repoPath = filepath.Clean(parsedURL.Path)
	}

	return buildGitRepoReference(cloneURL, repoURL, host, repoPath, ref)
}

func buildGitRepoReference(cloneURL, repoURL, host, repoPath, ref string) (*gitRepoReference, error) {
	repoPath = strings.TrimSpace(repoPath)
	repoPath = strings.TrimPrefix(repoPath, "/")
	repoPath = strings.TrimSuffix(repoPath, "/")
	repoPath = strings.TrimSuffix(repoPath, ".git")
	if repoPath == "" {
		return nil, errors.New("repo_url must include repository path")
	}

	parts := strings.Split(repoPath, "/")
	if host != localRepoHost && len(parts) < 2 {
		return nil, errors.New("repo_url must include owner and repository name")
	}
	repo := parts[len(parts)-1]
	owner := ""
	if len(parts) > 1 {
		owner = strings.Join(parts[:len(parts)-1], "/")
	}

	return &gitRepoReference{
		CloneURL: cloneURL,
		RepoURL:  strings.TrimSpace(repoURL),
		Host:     host,
		RepoPath: repoPath,
		Owner:    owner,
		Repo:     repo,
		Ref:      strings.TrimSpace(ref),
	}, nil
}

func runGitCommand(ctx context.Context, workdir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if workdir != "" {
		cmd.Dir = workdir
	}
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	return out, nil
}

func zipDirectory(rootDir, rootName string, maxArchiveBytes int64) ([]byte, error) {
	// Open a root-scoped handle so file reads inside the walk cannot be
	// redirected outside the repository via a symlink swap (TOCTOU).
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, fmt.Errorf("open repository tree: %w", err)
	}
	defer root.Close()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	var totalBytes int64
	totalLimit := effectiveGitSizeLimit(maxArchiveBytes)

	walkErr := filepath.WalkDir(rootDir, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(rootDir, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		rel = filepath.ToSlash(rel)
		if rel == ".git" || strings.HasPrefix(rel, ".git/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("repository contains unsupported symlink %q", rel)
		}
		if entry.IsDir() {
			return nil
		}

		cleanRel := safeRelPath(rel)
		if cleanRel == "" {
			return fmt.Errorf("invalid repository path %q", rel)
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if fileInfo.Size() > maxEntryBytes {
			return fmt.Errorf("file %q exceeds maximum size (%d > %d)", rel, fileInfo.Size(), maxEntryBytes)
		}
		totalBytes += fileInfo.Size()
		if totalBytes > totalLimit {
			return errors.New("repository exceeds maximum size")
		}

		handle, err := root.Open(filepath.FromSlash(cleanRel))
		if err != nil {
			return err
		}
		defer handle.Close()

		header, err := zip.FileInfoHeader(fileInfo)
		if err != nil {
			return err
		}
		header.Name = path.Join(rootName, cleanRel)
		header.Method = zip.Deflate

		dst, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, io.LimitReader(handle, maxEntryBytes+entryReadOverhead)); err != nil {
			return err
		}
		if maxArchiveBytes > 0 && int64(buf.Len()) > maxArchiveBytes {
			return errors.New("repository archive exceeds maximum size")
		}

		return nil
	})
	if walkErr != nil {
		_ = writer.Close()
		return nil, walkErr
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	if maxArchiveBytes > 0 && int64(buf.Len()) > maxArchiveBytes {
		return nil, errors.New("repository archive exceeds maximum size")
	}

	return buf.Bytes(), nil
}

func looksLikeLocalGitPath(repoURL string) bool {
	if filepath.IsAbs(repoURL) ||
		repoURL == "." ||
		strings.HasPrefix(repoURL, "./") ||
		strings.HasPrefix(repoURL, "../") {
		return true
	}

	if strings.Contains(repoURL, "://") || strings.Contains(repoURL, ":") {
		return false
	}

	info, err := os.Stat(repoURL)
	return err == nil && info.IsDir()
}

func effectiveGitSizeLimit(maxArchiveBytes int64) int64 {
	limit := int64(maxTotalBytes)
	if maxArchiveBytes > 0 && maxArchiveBytes < limit {
		return maxArchiveBytes
	}
	return limit
}

func ensureDirectorySizeLimit(rootDir string, maxBytes int64) error {
	if maxBytes <= 0 {
		return nil
	}

	var totalBytes int64
	walkErr := filepath.WalkDir(rootDir, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(rootDir, current)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".git" || strings.HasPrefix(rel, ".git/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.IsDir() {
			return nil
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		totalBytes += fileInfo.Size()
		if totalBytes > maxBytes {
			return fmt.Errorf("repository exceeds maximum size while reading %q", rel)
		}
		return nil
	})
	if walkErr != nil {
		return walkErr
	}
	return nil
}

func optionalTrimmedString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
