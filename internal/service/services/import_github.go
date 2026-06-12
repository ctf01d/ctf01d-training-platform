package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/storage"
)

type GithubImportRequest struct {
	RepoURL string
	Ref     string
	Subdir  string
}

type ImportResult struct {
	Service  *ServiceModel
	Warnings []string
}

type ImportQuerier interface {
	GetServiceByName(ctx context.Context, name string) (db.Service, error)
	CreateService(ctx context.Context, arg db.CreateServiceParams) (db.Service, error)
	UpdateService(ctx context.Context, arg db.UpdateServiceParams) (db.Service, error)
	GetServiceByID(ctx context.Context, id int64) (db.Service, error)
	SetServiceLocal(ctx context.Context, arg db.SetServiceLocalParams) (db.Service, error)
	SetCheckerLocal(ctx context.Context, arg db.SetCheckerLocalParams) (db.Service, error)
	SetArchiveURLs(ctx context.Context, arg db.SetArchiveURLsParams) (db.Service, error)
}

type ImportService struct {
	q              ImportQuerier
	store          storage.Storage
	maxUploadBytes int64
	httpClient     *http.Client
}

func NewImportService(q ImportQuerier, store storage.Storage, maxUploadBytes int64) *ImportService {
	return &ImportService{
		q:              q,
		store:          store,
		maxUploadBytes: maxUploadBytes,
		httpClient: &http.Client{
			Timeout: archiveDownloadTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return errors.New("too many redirects")
				}
				if req.URL.Hostname() == "" {
					return errors.New("redirect URL has no host")
				}
				return nil
			},
			Transport: &http.Transport{
				DialContext: ssrfSafeDialContext,
			},
		},
	}
}

var serviceNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (s *ImportService) ImportFromGithub(ctx context.Context, req GithubImportRequest, isAdmin bool) (*ImportResult, error) {
	if req.Subdir != "" {
		return nil, errs.NewValidationError(map[string]string{"subdir": "import is not yet supported"})
	}

	owner, repo, parsedRef, err := parseGitHubURL(req.RepoURL)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{"repo_url": "must be a valid GitHub repository URL"})
	}

	ref := req.Ref
	if ref == "" {
		ref = parsedRef
	}
	if ref == "" {
		ref = "main"
	}

	zipBytes, err := s.fetchRepoZip(ctx, owner, repo, ref)
	if err != nil {
		return nil, fmt.Errorf("downloading repository: %w", err)
	}

	bundleBytes, err := BuildBundle(zipBytes)
	if err != nil {
		return nil, fmt.Errorf("building bundle: %w", err)
	}

	meta, err := ExtractMetadata(bundleBytes)
	if err != nil {
		return nil, fmt.Errorf("extracting metadata: %w", err)
	}

	var warnings []string

	name := meta.Name
	if name == "" {
		name = repo
		warnings = append(warnings, "no name found in metadata, using repository name")
	}
	if !serviceNameRe.MatchString(name) {
		return nil, fmt.Errorf("invalid service name derived from import: %q", name)
	}

	archiveURL := githubArchiveURL(owner, repo, ref)

	var training json.RawMessage
	if meta.Ctf01dTraining != nil {
		training = meta.Ctf01dTraining
	} else {
		training = json.RawMessage("{}")
	}

	existing, err := s.q.GetServiceByName(ctx, name)
	if err == nil {
		if !isAdmin {
			return nil, fmt.Errorf("service with name %q already exists; only admins can update existing services via import", name)
		}
		return s.updateFromImport(ctx, existing.ID, name, bundleBytes, meta, archiveURL, training, isAdmin, warnings)
	}

	svc, err := s.q.CreateService(ctx, db.CreateServiceParams{
		Name:              name,
		PublicDescription: &meta.PublicDescription,
		Copyright:         &meta.Copyright,
		Public:            true,
		ServiceArchiveUrl: &archiveURL,
		Ctf01dTraining:    training,
		CheckStatus:       checkStatusUnknown,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return nil, fmt.Errorf("service with name %q already exists", name)
		}
		return nil, fmt.Errorf("creating service: %w", err)
	}

	result := s.saveBundleArchives(ctx, svc.ID, bundleBytes, isAdmin, warnings)
	return result, nil
}

func (s *ImportService) updateFromImport(ctx context.Context, id int64, name string, bundleBytes []byte, meta *BundleMetadata, archiveURL string, training json.RawMessage, isAdmin bool, warnings []string) (*ImportResult, error) {
	svc, err := s.q.UpdateService(ctx, db.UpdateServiceParams{
		ID:                id,
		Name:              name,
		PublicDescription: &meta.PublicDescription,
		Copyright:         &meta.Copyright,
		ServiceArchiveUrl: &archiveURL,
		Ctf01dTraining:    training,
		Public:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("updating service: %w", err)
	}

	result := s.saveBundleArchivesForSvc(ctx, svc, bundleBytes, isAdmin, warnings)
	return result, nil
}

func (s *ImportService) saveBundleArchives(ctx context.Context, id int64, bundleBytes []byte, isAdmin bool, warnings []string) *ImportResult {
	now := time.Now()
	svc := db.Service{ID: id}

	key := fmt.Sprintf("services/%d/service.zip", id)
	if _, err := s.store.Save(ctx, key, bytes.NewReader(bundleBytes)); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to save service archive: %v", err))
	} else {
		size, err := int32Len(len(bundleBytes))
		if err != nil {
			warnings = append(warnings, err.Error())
		} else {
			sha := computeSHA256Hex(bundleBytes)
			path := key
			updated, err := s.q.SetServiceLocal(ctx, db.SetServiceLocalParams{
				ID:                  id,
				ServiceLocalPath:    &path,
				ServiceLocalSize:    &size,
				ServiceLocalSha256:  &sha,
				ServiceDownloadedAt: pgtypeTz(now),
			})
			if err == nil {
				svc = updated
			}
		}
	}

	checkerBytes := extractCheckerFromBundle(bundleBytes)
	if len(checkerBytes) > 0 {
		ckKey := fmt.Sprintf("services/%d/checker.zip", id)
		if _, err := s.store.Save(ctx, ckKey, bytes.NewReader(checkerBytes)); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to save checker archive: %v", err))
		} else {
			ckSize, err := int32Len(len(checkerBytes))
			if err != nil {
				warnings = append(warnings, err.Error())
			} else {
				ckSha := computeSHA256Hex(checkerBytes)
				ckPath := ckKey
				updated, err := s.q.SetCheckerLocal(ctx, db.SetCheckerLocalParams{
					ID:                  id,
					CheckerLocalPath:    &ckPath,
					CheckerLocalSize:    &ckSize,
					CheckerLocalSha256:  &ckSha,
					CheckerDownloadedAt: pgtypeTz(now),
				})
				if err == nil {
					svc = updated
				}
			}
		}
	}

	model := fromDB(svc, isAdmin)
	return &ImportResult{
		Service:  &model,
		Warnings: warnings,
	}
}

func (s *ImportService) saveBundleArchivesForSvc(ctx context.Context, svc db.Service, bundleBytes []byte, isAdmin bool, warnings []string) *ImportResult {
	return s.saveBundleArchives(ctx, svc.ID, bundleBytes, isAdmin, warnings)
}

func (s *ImportService) ImportFromZip(ctx context.Context, zipBytes []byte, isAdmin bool) (*ImportResult, error) {
	if err := validateZipBytes(zipBytes); err != nil {
		return nil, fmt.Errorf("invalid zip: %w", err)
	}

	bundleBytes, err := BuildBundle(zipBytes)
	if err != nil {
		return nil, fmt.Errorf("building bundle: %w", err)
	}

	meta, err := ExtractMetadata(bundleBytes)
	if err != nil {
		return nil, fmt.Errorf("extracting metadata: %w", err)
	}

	var warnings []string

	name := meta.Name
	if name == "" {
		return nil, errors.New("could not determine service name from zip")
	}
	if !serviceNameRe.MatchString(name) {
		return nil, fmt.Errorf("invalid service name derived from import: %q", name)
	}

	var training json.RawMessage
	if meta.Ctf01dTraining != nil {
		training = meta.Ctf01dTraining
	} else {
		training = json.RawMessage("{}")
	}

	svc, err := s.q.CreateService(ctx, db.CreateServiceParams{
		Name:              name,
		PublicDescription: &meta.PublicDescription,
		Copyright:         &meta.Copyright,
		Public:            true,
		Ctf01dTraining:    training,
		CheckStatus:       checkStatusUnknown,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return nil, fmt.Errorf("service with name %q already exists", name)
		}
		return nil, fmt.Errorf("creating service: %w", err)
	}

	result := s.saveBundleArchives(ctx, svc.ID, bundleBytes, isAdmin, warnings)
	return result, nil
}

func (s *ImportService) fetchRepoZip(ctx context.Context, owner, repo, ref string) ([]byte, error) {
	refPaths := []string{
		"refs/heads/" + ref,
		"refs/tags/" + ref,
		ref,
	}
	for _, refPath := range refPaths {
		url := codeloadURL(owner, repo, refPath)
		data, err := s.downloadZipBytes(ctx, url)
		if err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("failed to download repository archive %s/%s@%s", owner, repo, ref)
}

func (s *ImportService) downloadZipBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, s.maxUploadBytes+uploadLimitOverhead))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > s.maxUploadBytes {
		return nil, errors.New("archive exceeds maximum size")
	}
	if err := validateZipBytes(data); err != nil {
		return nil, err
	}
	return data, nil
}

func parseGitHubURL(repoURL string) (owner, repo, ref string, err error) {
	repoURL = strings.TrimSpace(repoURL)
	if !strings.HasPrefix(repoURL, "https://") {
		return "", "", "", errors.New("invalid URL: must use https")
	}
	schemeEnd := strings.Index(repoURL, "://")
	if schemeEnd < 0 {
		return "", "", "", errors.New("invalid URL")
	}
	rest := repoURL[schemeEnd+3:]
	hostEnd := strings.Index(rest, "/")
	if hostEnd < 0 {
		return "", "", "", errors.New("invalid GitHub URL: expected /owner/repo")
	}
	host := rest[:hostEnd]
	if host != "github.com" {
		return "", "", "", errors.New("not a github.com URL")
	}
	pathPart := rest[hostEnd+1:]
	parts := strings.Split(pathPart, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", errors.New("invalid GitHub URL: expected /owner/repo")
	}
	owner = parts[0]
	repo = parts[1]
	repo = strings.TrimSuffix(repo, ".git")
	if len(parts) >= 4 && parts[2] == "tree" && parts[3] != "" {
		ref = parts[3]
	}
	return owner, repo, ref, nil
}

var githubArchiveURL = func(owner, repo, ref string) string {
	return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip", owner, repo, ref)
}

var codeloadURL = func(owner, repo, refPath string) string {
	return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/%s", owner, repo, refPath)
}
