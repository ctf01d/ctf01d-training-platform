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

type ImportValidationItem struct {
	ID      string
	Title   string
	Status  string
	Message string
}

type ImportPreview struct {
	Source                 string
	Valid                  bool
	ServiceName            string
	RepositoryOwner        string
	RepositoryName         string
	ExpectedRepositoryName string
	RootDirectory          string
	ServiceDirectory       string
	CheckerDirectory       string
	HasDevDirectory        bool
	ExistingServiceID      *int64
	Requirements           []ImportValidationItem
	Warnings               []string
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

func (s *ImportService) PreviewFromGithub(ctx context.Context, req GithubImportRequest, isAdmin bool) (*ImportPreview, error) {
	if req.Subdir != "" {
		return nil, errs.NewValidationError(map[string]string{"subdir": "import is not yet supported"})
	}

	owner, repo, parsedRef, err := parseGitHubURL(req.RepoURL)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldRepoURL: "must be a valid GitHub repository URL"})
	}

	ref := req.Ref
	if ref == "" {
		ref = parsedRef
	}
	if ref == "" {
		ref = defaultGitRef
	}

	zipBytes, err := s.fetchRepoZip(ctx, owner, repo, ref)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldRepoURL: fmt.Sprintf("downloading repository: %v", err)})
	}

	source := importSourceInfo{Source: sourceGithub, Owner: owner, Repo: repo}
	return s.previewArchive(ctx, zipBytes, source, isAdmin)
}

func (s *ImportService) PreviewFromZipUpload(ctx context.Context, zipBytes []byte, isAdmin bool) (*ImportPreview, error) {
	if len(zipBytes) == 0 {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: "file is required"})
	}
	if err := validateZipBytes(zipBytes); err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: err.Error()})
	}
	return s.previewArchive(ctx, zipBytes, importSourceInfo{Source: sourceZip}, isAdmin)
}

var serviceNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (s *ImportService) previewArchive(ctx context.Context, zipBytes []byte, source importSourceInfo, isAdmin bool) (*ImportPreview, error) {
	layout, err := inspectSourceLayoutFromBytes(zipBytes, source)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: err.Error()})
	}

	var meta *BundleMetadata
	var bundleErr error
	bundleBytes, err := BuildBundle(zipBytes)
	if err != nil {
		bundleErr = err
	} else {
		meta, err = ExtractMetadata(bundleBytes)
		if err != nil {
			bundleErr = err
		}
	}

	serviceName := resolveImportServiceName(meta, layout, source)
	preview := buildImportPreview(layout, source, serviceName)
	if bundleErr != nil {
		preview.Requirements = append(preview.Requirements, ImportValidationItem{
			ID:      "archive",
			Title:   "Archive can be normalized",
			Status:  "error",
			Message: bundleErr.Error(),
		})
	}

	s.addDuplicatePreviewStatus(ctx, preview, source, serviceName, isAdmin)
	finalizeImportPreview(preview)
	return preview, nil
}

func buildImportPreview(layout sourceLayout, source importSourceInfo, serviceName string) *ImportPreview {
	preview := &ImportPreview{
		Source:                 source.Source,
		ServiceName:            serviceName,
		RepositoryOwner:        source.Owner,
		RepositoryName:         source.Repo,
		ExpectedRepositoryName: serviceRepoPattern,
		RootDirectory:          layout.RootName,
		ServiceDirectory:       layout.ServiceDir,
		CheckerDirectory:       layout.CheckerDir,
		HasDevDirectory:        layout.HasDevDir,
		Requirements:           make([]ImportValidationItem, 0, previewRequirementsCapacity),
		Warnings:               []string{},
	}

	switch {
	case source.Source == sourceGithub:
		if strings.EqualFold(source.Owner, expectedGithubOwner) {
			preview.addRequirement("github_owner", "GitHub organization", "ok", "repository is in github.com/"+expectedGithubOwner)
		} else {
			preview.addRequirement("github_owner", "GitHub organization", "error", "repository must be in github.com/"+expectedGithubOwner)
		}
		if serviceIDFromRepo(source.Repo) != "" {
			preview.addRequirement("repository_name", "Repository name", "ok", "matches "+serviceRepoPattern)
		} else {
			preview.addRequirement("repository_name", "Repository name", "error", "must match "+serviceRepoPattern)
		}
	case layout.RootName != "" && serviceIDFromRepo(layout.RootName) != "":
		preview.addRequirement("repository_name", "Repository name", "ok", "archive root looks like "+serviceRepoPattern)
	default:
		preview.addRequirement("repository_name", "Repository name", "warning", "ZIP upload cannot prove the final GitHub repository name")
	}

	switch {
	case serviceName != "" && serviceNameRe.MatchString(serviceName):
		preview.addRequirement("service_id", "Service ID", "ok", "will be imported as "+serviceName)
	case serviceName != "":
		preview.addRequirement("service_id", "Service ID", "error", "derived service name must match [a-zA-Z0-9_-]+")
	default:
		preview.addRequirement("service_id", "Service ID", "error", "service ID was not found in repository name, checker directory, or metadata")
	}

	if layout.HasRootReadme {
		preview.addRequirement("root_readme", "README.md", "ok", "root README.md is present")
	} else {
		preview.addRequirement("root_readme", "README.md", "error", "root README.md is required")
	}

	switch {
	case layout.HasNewService:
		preview.addRequirement("vuln_service", "vuln-service", "ok", "service directory is present")
	case layout.HasOldService:
		preview.addRequirement("vuln_service", "vuln-service", "error", "legacy service/ was found, but vuln-service/ is required")
	default:
		preview.addRequirement("vuln_service", "vuln-service", "error", "vuln-service/ directory is required")
	}

	if layout.ComposeFile != "" {
		preview.addRequirement("docker_compose", "docker compose", "ok", layout.ComposeFile+" is present")
	} else {
		preview.addRequirement("docker_compose", "docker compose", "error", "vuln-service must contain docker-compose.yml, docker-compose.yaml, compose.yml, or compose.yaml")
	}

	expectedCheckerDir := ""
	if serviceName != "" && serviceNameRe.MatchString(serviceName) {
		expectedCheckerDir = checkerDirPrefix + serviceName
	}
	switch {
	case expectedCheckerDir != "" && layout.CheckerDir == expectedCheckerDir:
		preview.addRequirement("checker", "checker_<idservice>", "ok", layout.CheckerDir+" is present")
	case len(layout.CheckerDirs) > 0 && expectedCheckerDir != "":
		preview.addRequirement("checker", "checker_<idservice>", "error", "checker directory must be named "+expectedCheckerDir)
	case len(layout.CheckerDirs) > 0:
		preview.addRequirement("checker", "checker_<idservice>", "ok", layout.CheckerDir+" is present")
	case layout.HasOldChecker && expectedCheckerDir != "":
		preview.addRequirement("checker", "checker_<idservice>", "error", "legacy checker/ was found, but "+expectedCheckerDir+" is required")
	case layout.HasOldChecker:
		preview.addRequirement("checker", "checker_<idservice>", "error", "legacy checker/ was found, but checker_<idservice>/ is required")
	case expectedCheckerDir != "":
		preview.addRequirement("checker", "checker_<idservice>", "error", expectedCheckerDir+" directory is required")
	default:
		preview.addRequirement("checker", "checker_<idservice>", "error", "checker_<idservice> directory is required")
	}

	if layout.HasWriteups {
		preview.addRequirement("writeups", "writeups", "ok", "writeups/ is present")
	} else {
		preview.addRequirement("writeups", "writeups", "error", "writeups/ directory is required")
	}

	if layout.HasExploits {
		preview.addRequirement("exploits", "exploits", "ok", "exploits/ is present")
	} else {
		preview.addRequirement("exploits", "exploits", "error", "exploits/ directory is required")
	}

	if layout.HasDevDir {
		preview.addRequirement("vuln_service_dev", "vuln-service_dev", "ok", "optional build directory is present")
	} else {
		preview.addRequirement("vuln_service_dev", "vuln-service_dev", "warning", "optional directory is not present")
	}

	return preview
}

func (p *ImportPreview) addRequirement(id, title, status, message string) {
	p.Requirements = append(p.Requirements, ImportValidationItem{
		ID:      id,
		Title:   title,
		Status:  status,
		Message: message,
	})
}

func (s *ImportService) addDuplicatePreviewStatus(ctx context.Context, preview *ImportPreview, source importSourceInfo, serviceName string, isAdmin bool) {
	if serviceName == "" || !serviceNameRe.MatchString(serviceName) {
		return
	}
	existing, err := s.q.GetServiceByName(ctx, serviceName)
	if err != nil {
		preview.addRequirement("duplicate", "Existing service", "ok", "no service with this name exists")
		return
	}
	preview.ExistingServiceID = &existing.ID
	if source.Source == sourceGithub && isAdmin {
		preview.addRequirement("duplicate", "Existing service", "warning", "existing service will be updated by GitHub import")
		return
	}
	preview.addRequirement("duplicate", "Existing service", "error", "service with this name already exists")
}

func finalizeImportPreview(preview *ImportPreview) {
	preview.Valid = true
	preview.Warnings = preview.Warnings[:0]
	for _, item := range preview.Requirements {
		switch item.Status {
		case "error":
			preview.Valid = false
		case "warning":
			preview.Warnings = append(preview.Warnings, item.Message)
		}
	}
}

func (s *ImportService) ImportFromGithub(ctx context.Context, req GithubImportRequest, isAdmin bool) (*ImportResult, error) {
	if req.Subdir != "" {
		return nil, errs.NewValidationError(map[string]string{"subdir": "import is not yet supported"})
	}

	owner, repo, parsedRef, err := parseGitHubURL(req.RepoURL)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldRepoURL: "must be a valid GitHub repository URL"})
	}

	ref := req.Ref
	if ref == "" {
		ref = parsedRef
	}
	if ref == "" {
		ref = defaultGitRef
	}

	zipBytes, err := s.fetchRepoZip(ctx, owner, repo, ref)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldRepoURL: fmt.Sprintf("downloading repository: %v", err)})
	}
	source := importSourceInfo{Source: sourceGithub, Owner: owner, Repo: repo}
	layout, err := inspectSourceLayoutFromBytes(zipBytes, source)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: err.Error()})
	}

	bundleBytes, err := BuildBundle(zipBytes)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: fmt.Sprintf("building bundle: %v", err)})
	}

	meta, err := ExtractMetadata(bundleBytes)
	if err != nil {
		return nil, fmt.Errorf("extracting metadata: %w", err)
	}

	var warnings []string
	preview := buildImportPreview(layout, source, resolveImportServiceName(meta, layout, source))
	finalizeImportPreview(preview)
	warnings = append(warnings, preview.Warnings...)
	if !preview.Valid {
		warnings = append(warnings, "repository layout does not satisfy all Cybersibir service requirements")
	}

	name := resolveImportServiceName(meta, layout, source)
	if name == "" {
		name = repo
		warnings = append(warnings, "no name found in metadata, using repository name")
	}
	if !serviceNameRe.MatchString(name) {
		return nil, errs.NewValidationError(map[string]string{fieldName: fmt.Sprintf("invalid service name derived from import: %q", name)})
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
			return nil, errs.ErrConflict
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
			return nil, errs.ErrConflict
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
		return nil, errs.NewValidationError(map[string]string{fieldArchive: fmt.Sprintf("invalid zip: %v", err)})
	}

	source := importSourceInfo{Source: sourceZip}
	layout, err := inspectSourceLayoutFromBytes(zipBytes, source)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: err.Error()})
	}

	bundleBytes, err := BuildBundle(zipBytes)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: fmt.Sprintf("building bundle: %v", err)})
	}

	meta, err := ExtractMetadata(bundleBytes)
	if err != nil {
		return nil, fmt.Errorf("extracting metadata: %w", err)
	}

	var warnings []string
	preview := buildImportPreview(layout, source, resolveImportServiceName(meta, layout, source))
	finalizeImportPreview(preview)
	warnings = append(warnings, preview.Warnings...)
	if !preview.Valid {
		warnings = append(warnings, "repository layout does not satisfy all Cybersibir service requirements")
	}

	name := resolveImportServiceName(meta, layout, source)
	if name == "" {
		return nil, errs.NewValidationError(map[string]string{fieldName: "could not determine service name from zip"})
	}
	if !serviceNameRe.MatchString(name) {
		return nil, errs.NewValidationError(map[string]string{fieldName: fmt.Sprintf("invalid service name derived from import: %q", name)})
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
			return nil, errs.ErrConflict
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
