package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/storage"
)

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
	ApplyServiceImportMetadata(ctx context.Context, arg db.ApplyServiceImportMetadataParams) (db.Service, error)
	UpdateService(ctx context.Context, arg db.UpdateServiceParams) (db.Service, error)
	GetServiceByID(ctx context.Context, id int64) (db.Service, error)
	SetServiceLocal(ctx context.Context, arg db.SetServiceLocalParams) (db.Service, error)
	SetCheckerLocal(ctx context.Context, arg db.SetCheckerLocalParams) (db.Service, error)
	SetGitSource(ctx context.Context, arg db.SetGitSourceParams) (db.Service, error)
	SetGitSyncState(ctx context.Context, arg db.SetGitSyncStateParams) (db.Service, error)
}

type ImportService struct {
	q              ImportQuerier
	store          storage.Storage
	maxUploadBytes int64
	gitFetcher     gitArchiveFetcher
}

type preparedImport struct {
	Name        string
	Layout      sourceLayout
	Meta        *BundleMetadata
	BundleBytes []byte
	Preview     *ImportPreview
	Training    json.RawMessage
}

func NewImportService(q ImportQuerier, store storage.Storage, maxUploadBytes int64) *ImportService {
	return &ImportService{
		q:              q,
		store:          store,
		maxUploadBytes: maxUploadBytes,
		gitFetcher:     newExecGitArchiveFetcher(maxUploadBytes),
	}
}

func (s *ImportService) PreviewFromGit(ctx context.Context, req GitImportRequest, isAdmin bool) (*ImportPreview, error) {
	if !isAdmin {
		return nil, errs.ErrForbidden
	}

	fetched, err := s.gitFetcher.Fetch(ctx, req)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldRepoURL: err.Error()})
	}

	return s.previewArchive(ctx, fetched.ZipBytes, fetched.Source, isAdmin, nil)
}

func (s *ImportService) PreviewFromZipUpload(ctx context.Context, zipBytes []byte, isAdmin bool) (*ImportPreview, error) {
	if len(zipBytes) == 0 {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: "file is required"})
	}
	if err := validateZipBytes(zipBytes); err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: err.Error()})
	}

	return s.previewArchive(ctx, zipBytes, importSourceInfo{Source: sourceZip}, isAdmin, nil)
}

var serviceNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (s *ImportService) previewArchive(
	ctx context.Context,
	zipBytes []byte,
	source importSourceInfo,
	isAdmin bool,
	currentID *int64,
) (*ImportPreview, error) {
	prepared, err := s.prepareImport(ctx, zipBytes, source, isAdmin, currentID)
	if err != nil {
		return nil, err
	}
	return prepared.Preview, nil
}

func buildImportPreview(layout sourceLayout, source importSourceInfo, meta *BundleMetadata, serviceName string) *ImportPreview {
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
		Requirements:           make([]ImportValidationItem, 0, previewRequirementsCapacity+manifestExtraRequirements),
		Warnings:               []string{},
	}

	switch {
	case source.Source == sourceGit && source.Host != "":
		location := source.Host
		if source.Path != "" {
			location += "/" + source.Path
		}
		preview.addRequirement("repository_source", "Git repository", "ok", location)
	case source.Source == sourceGit && source.Repo != "":
		preview.addRequirement("repository_source", "Git repository", "ok", source.Repo)
	default:
		preview.addRequirement("repository_source", "Import source", "ok", "ZIP upload")
	}

	switch {
	case source.Repo != "" && serviceIDFromRepo(source.Repo) != "":
		preview.addRequirement("repository_name", "Repository name", "ok", "matches "+serviceRepoPattern)
	case layout.RootName != "" && serviceIDFromRepo(layout.RootName) != "":
		preview.addRequirement("repository_name", "Repository name", "ok", "archive root looks like "+serviceRepoPattern)
	case source.Source == sourceGit:
		preview.addRequirement("repository_name", "Repository name", "warning", "repository name does not follow "+serviceRepoPattern)
	default:
		preview.addRequirement("repository_name", "Repository name", "warning", "ZIP upload cannot prove the final repository name")
	}

	switch {
	case serviceName != "" && serviceNameRe.MatchString(serviceName):
		preview.addRequirement("service_id", "Service ID", "ok", "will be imported as "+serviceName)
	case serviceName != "":
		preview.addRequirement("service_id", "Service ID", "error", "derived service name must match [a-zA-Z0-9_-]+")
	default:
		preview.addRequirement("service_id", "Service ID", "error", "service ID was not found in repository layout or metadata")
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
	addCheckerRequirement(preview, layout, expectedCheckerDir)

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

	addManifestRequirements(preview, meta)

	return preview
}

func addCheckerRequirement(preview *ImportPreview, layout sourceLayout, expectedCheckerDir string) {
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
}

func addManifestRequirements(preview *ImportPreview, meta *BundleMetadata) {
	if meta == nil || meta.Manifest == nil {
		preview.addRequirement("service_manifest", serviceManifestYAML, "warning", "service manifest is not present; checker settings will not be synchronized")
		return
	}

	preview.addRequirement("service_manifest", serviceManifestYAML, "ok", "service manifest is present")
	switch {
	case meta.Manifest.ID == "":
		preview.addRequirement("manifest_id", "manifest id", "error", "manifest must define checker-config-*/id")
	case serviceNameRe.MatchString(meta.Manifest.ID):
		preview.addRequirement("manifest_id", "manifest id", "ok", "manifest id is "+meta.Manifest.ID)
	default:
		preview.addRequirement("manifest_id", "manifest id", "error", "manifest id must match [a-zA-Z0-9_-]+")
	}

	switch {
	case meta.Manifest.ScriptPath == "":
		preview.addRequirement("checker_script", "checker script_path", "error", "manifest must define checker-config-*/script_path")
	case meta.CheckerScriptOK:
		preview.addRequirement("checker_script", "checker script_path", "ok", meta.Manifest.ScriptPath+" is present")
	default:
		preview.addRequirement("checker_script", "checker script_path", "error", meta.Manifest.ScriptPath+" was not found in checker/")
	}
}

func (p *ImportPreview) addRequirement(id, title, status, message string) {
	p.Requirements = append(p.Requirements, ImportValidationItem{
		ID:      id,
		Title:   title,
		Status:  status,
		Message: message,
	})
}

func (s *ImportService) addDuplicatePreviewStatus(
	ctx context.Context,
	preview *ImportPreview,
	source importSourceInfo,
	serviceName string,
	isAdmin bool,
	currentID *int64,
) {
	if serviceName == "" || !serviceNameRe.MatchString(serviceName) {
		return
	}

	existing, err := s.q.GetServiceByName(ctx, serviceName)
	if err != nil {
		preview.addRequirement("duplicate", "Existing service", "ok", "no service with this name exists")
		return
	}

	preview.ExistingServiceID = &existing.ID
	if currentID != nil && existing.ID == *currentID {
		preview.addRequirement("duplicate", "Existing service", "ok", "this service will be synchronized")
		return
	}
	if source.Source == sourceGit && isAdmin {
		preview.addRequirement("duplicate", "Existing service", "warning", "existing service will be updated by git import")
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

func (s *ImportService) ImportFromGit(ctx context.Context, req GitImportRequest, isAdmin bool) (*ImportResult, error) {
	if !isAdmin {
		return nil, errs.ErrForbidden
	}

	fetched, err := s.gitFetcher.Fetch(ctx, req)
	if err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldRepoURL: err.Error()})
	}

	prepared, err := s.prepareImport(ctx, fetched.ZipBytes, fetched.Source, isAdmin, nil)
	if err != nil {
		return nil, err
	}
	if err := validatePreparedImport(prepared.Preview, fieldRepoURL); err != nil {
		return nil, err
	}

	existing, err := s.q.GetServiceByName(ctx, prepared.Name)
	if err == nil {
		if !isAdmin {
			return nil, errs.ErrConflict
		}
		return s.updateServiceFromGitImport(ctx, existing, fetched, prepared, isAdmin)
	}

	svc, err := s.q.CreateService(ctx, db.CreateServiceParams{
		Name:              prepared.Name,
		PublicDescription: optionalImportedString(prepared.Meta.PublicDescription),
		Author:            metaAuthorPtr(prepared.Meta),
		Copyright:         optionalImportedString(prepared.Meta.Copyright),
		Public:            true,
		Ctf01dTraining:    prepared.Training,
		CheckStatus:       checkStatusUnknown,
		SourceKind:        sourceGit,
		GitRepoUrl:        optionalImportedString(fetched.RepoURL),
		GitRef:            optionalImportedString(fetched.Ref),
		GitSubdir:         optionalImportedString(fetched.Subdir),
		GitSyncStatus:     syncStatusUnknown,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return nil, errs.ErrConflict
		}
		return nil, fmt.Errorf("creating service: %w", err)
	}

	result, err := s.saveBundleArchives(ctx, svc.ID, prepared.BundleBytes, isAdmin, prepared.Preview.Warnings)
	if err != nil {
		return nil, err
	}
	return s.markGitSyncSuccess(ctx, result, fetched.Commit, isAdmin)
}

func (s *ImportService) SyncFromGit(ctx context.Context, id int64, isAdmin bool) (*ServiceModel, error) {
	if !isAdmin {
		return nil, errs.ErrForbidden
	}

	current, err := s.q.GetServiceByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if current.SourceKind != sourceGit || current.GitRepoUrl == nil || strings.TrimSpace(*current.GitRepoUrl) == "" {
		return nil, errs.NewValidationError(map[string]string{fieldRepoURL: "git source is not configured"})
	}

	req := GitImportRequest{
		RepoURL: *current.GitRepoUrl,
	}
	if current.GitRef != nil {
		req.Ref = *current.GitRef
	}
	if current.GitSubdir != nil {
		req.Subdir = *current.GitSubdir
	}

	fetched, err := s.gitFetcher.Fetch(ctx, req)
	if err != nil {
		s.markGitSyncFailureAndLog(ctx, id, syncFailureMessage(err))
		return nil, errs.NewValidationError(map[string]string{fieldRepoURL: err.Error()})
	}

	prepared, err := s.prepareImport(ctx, fetched.ZipBytes, fetched.Source, isAdmin, &id)
	if err != nil {
		s.markGitSyncFailureAndLog(ctx, id, syncFailureMessage(err))
		return nil, err
	}
	if err := validatePreparedImport(prepared.Preview, fieldRepoURL); err != nil {
		s.markGitSyncFailureAndLog(ctx, id, syncFailureMessage(err))
		return nil, err
	}
	syncedName := current.Name
	if prepared.Name != "" && serviceNameRe.MatchString(prepared.Name) {
		syncedName = prepared.Name
	}

	if syncedName != current.Name {
		other, err := s.q.GetServiceByName(ctx, syncedName)
		if err == nil && other.ID != id {
			conflictErr := errs.ErrConflict
			s.markGitSyncFailureAndLog(ctx, id, syncFailureMessage(conflictErr))
			return nil, conflictErr
		}
	}

	svc, err := s.q.ApplyServiceImportMetadata(ctx, db.ApplyServiceImportMetadataParams{
		ID:                id,
		Name:              syncedName,
		PublicDescription: optionalImportedString(prepared.Meta.PublicDescription),
		Author:            metaAuthorPtr(prepared.Meta),
		Copyright:         optionalImportedString(prepared.Meta.Copyright),
		Ctf01dTraining:    prepared.Training,
	})
	if err != nil {
		s.markGitSyncFailureAndLog(ctx, id, syncFailureMessage(err))
		return nil, fmt.Errorf("updating service: %w", err)
	}

	result, err := s.saveBundleArchivesForSvc(ctx, svc, prepared.BundleBytes, isAdmin, prepared.Preview.Warnings)
	if err != nil {
		s.markGitSyncFailureAndLog(ctx, id, syncFailureMessage(err))
		return nil, err
	}
	result, err = s.markGitSyncSuccess(ctx, result, fetched.Commit, isAdmin)
	if err != nil {
		return nil, err
	}

	return result.Service, nil
}

func (s *ImportService) ImportFromZip(ctx context.Context, zipBytes []byte, isAdmin bool) (*ImportResult, error) {
	if err := validateZipBytes(zipBytes); err != nil {
		return nil, errs.NewValidationError(map[string]string{fieldArchive: fmt.Sprintf("invalid zip: %v", err)})
	}

	prepared, err := s.prepareImport(ctx, zipBytes, importSourceInfo{Source: sourceZip}, isAdmin, nil)
	if err != nil {
		return nil, err
	}
	if prepared.Name == "" {
		return nil, errs.NewValidationError(map[string]string{fieldName: "could not determine service name from zip"})
	}
	if !serviceNameRe.MatchString(prepared.Name) {
		return nil, errs.NewValidationError(map[string]string{fieldName: fmt.Sprintf("invalid service name derived from import: %q", prepared.Name)})
	}
	if err := validatePreparedImport(prepared.Preview, fieldArchive); err != nil {
		return nil, err
	}

	svc, err := s.q.CreateService(ctx, db.CreateServiceParams{
		Name:              prepared.Name,
		PublicDescription: optionalImportedString(prepared.Meta.PublicDescription),
		Author:            metaAuthorPtr(prepared.Meta),
		Copyright:         optionalImportedString(prepared.Meta.Copyright),
		Public:            true,
		Ctf01dTraining:    prepared.Training,
		CheckStatus:       checkStatusUnknown,
		SourceKind:        sourceZip,
		GitSyncStatus:     syncStatusUnknown,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return nil, errs.ErrConflict
		}
		return nil, fmt.Errorf("creating service: %w", err)
	}

	return s.saveBundleArchives(ctx, svc.ID, prepared.BundleBytes, isAdmin, prepared.Preview.Warnings)
}

func (s *ImportService) prepareImport(
	ctx context.Context,
	zipBytes []byte,
	source importSourceInfo,
	isAdmin bool,
	currentID *int64,
) (*preparedImport, error) {
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

	name := resolveImportServiceName(meta, layout, source)
	preview := buildImportPreview(layout, source, meta, name)
	s.addDuplicatePreviewStatus(ctx, preview, source, name, isAdmin, currentID)
	finalizeImportPreview(preview)

	training := meta.Ctf01dTraining
	if training == nil {
		training = json.RawMessage("{}")
	}

	return &preparedImport{
		Name:        name,
		Layout:      layout,
		Meta:        meta,
		BundleBytes: bundleBytes,
		Preview:     preview,
		Training:    training,
	}, nil
}

func validatePreparedImport(preview *ImportPreview, field string) error {
	if preview.Valid {
		return nil
	}

	var messages []string
	for _, item := range preview.Requirements {
		if item.Status == "error" {
			messages = append(messages, item.Message)
		}
	}
	if len(messages) == 0 {
		messages = append(messages, "import validation failed")
	}

	return errs.NewValidationError(map[string]string{field: strings.Join(messages, "; ")})
}

func syncFailureMessage(err error) string {
	if err == nil {
		return ""
	}

	var validationErr *errs.ValidationError
	if errors.As(err, &validationErr) && len(validationErr.Fields) > 0 {
		keys := make([]string, 0, len(validationErr.Fields))
		for key := range validationErr.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			value := strings.TrimSpace(validationErr.Fields[key])
			if value == "" {
				continue
			}
			if key == fieldRepoURL || key == fieldArchive {
				parts = append(parts, value)
				continue
			}
			parts = append(parts, key+": "+value)
		}
		if len(parts) > 0 {
			return strings.Join(parts, "; ")
		}
	}

	return err.Error()
}

func optionalImportedString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (s *ImportService) updateServiceFromGitImport(
	ctx context.Context,
	current db.Service,
	fetched *fetchedGitRepo,
	prepared *preparedImport,
	isAdmin bool,
) (*ImportResult, error) {
	if _, err := s.q.ApplyServiceImportMetadata(ctx, db.ApplyServiceImportMetadataParams{
		ID:                current.ID,
		Name:              prepared.Name,
		PublicDescription: optionalImportedString(prepared.Meta.PublicDescription),
		Author:            metaAuthorPtr(prepared.Meta),
		Copyright:         optionalImportedString(prepared.Meta.Copyright),
		Ctf01dTraining:    prepared.Training,
	}); err != nil {
		return nil, fmt.Errorf("updating service: %w", err)
	}

	svc, err := s.q.SetGitSource(ctx, db.SetGitSourceParams{
		ID:            current.ID,
		SourceKind:    sourceGit,
		GitRepoUrl:    optionalImportedString(fetched.RepoURL),
		GitRef:        optionalImportedString(fetched.Ref),
		GitSubdir:     optionalImportedString(fetched.Subdir),
		GitSyncStatus: syncStatusUnknown,
	})
	if err != nil {
		return nil, fmt.Errorf("updating git source: %w", err)
	}

	result, err := s.saveBundleArchivesForSvc(ctx, svc, prepared.BundleBytes, isAdmin, prepared.Preview.Warnings)
	if err != nil {
		return nil, err
	}
	return s.markGitSyncSuccess(ctx, result, fetched.Commit, isAdmin)
}

func (s *ImportService) saveBundleArchives(
	ctx context.Context,
	id int64,
	bundleBytes []byte,
	isAdmin bool,
	warnings []string,
) (*ImportResult, error) {
	now := time.Now()

	key := fmt.Sprintf("services/%d/service.zip", id)
	if _, err := s.store.Save(ctx, key, bytes.NewReader(bundleBytes)); err != nil {
		return nil, fmt.Errorf("saving service archive: %w", err)
	}

	size, err := int32Len(len(bundleBytes))
	if err != nil {
		return nil, err
	}
	sha := computeSHA256Hex(bundleBytes)
	path := key
	svc, err := s.q.SetServiceLocal(ctx, db.SetServiceLocalParams{
		ID:                  id,
		ServiceLocalPath:    &path,
		ServiceLocalSize:    &size,
		ServiceLocalSha256:  &sha,
		ServiceDownloadedAt: pgtypeTz(now),
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	checkerBytes := extractCheckerFromBundle(bundleBytes)
	if len(checkerBytes) > 0 {
		ckKey := fmt.Sprintf("services/%d/checker.zip", id)
		if _, err := s.store.Save(ctx, ckKey, bytes.NewReader(checkerBytes)); err != nil {
			return nil, fmt.Errorf("saving checker archive: %w", err)
		}

		ckSize, err := int32Len(len(checkerBytes))
		if err != nil {
			return nil, err
		}
		ckSha := computeSHA256Hex(checkerBytes)
		ckPath := ckKey
		svc, err = s.q.SetCheckerLocal(ctx, db.SetCheckerLocalParams{
			ID:                  id,
			CheckerLocalPath:    &ckPath,
			CheckerLocalSize:    &ckSize,
			CheckerLocalSha256:  &ckSha,
			CheckerDownloadedAt: pgtypeTz(now),
		})
		if err != nil {
			return nil, mapDBError(err)
		}
	}

	model := fromDB(svc, isAdmin)
	return &ImportResult{
		Service:  &model,
		Warnings: warnings,
	}, nil
}

func (s *ImportService) saveBundleArchivesForSvc(
	ctx context.Context,
	svc db.Service,
	bundleBytes []byte,
	isAdmin bool,
	warnings []string,
) (*ImportResult, error) {
	return s.saveBundleArchives(ctx, svc.ID, bundleBytes, isAdmin, warnings)
}

func (s *ImportService) markGitSyncSuccess(
	ctx context.Context,
	result *ImportResult,
	commit string,
	isAdmin bool,
) (*ImportResult, error) {
	if result == nil || result.Service == nil {
		return result, nil
	}

	svc, err := s.q.SetGitSyncState(ctx, db.SetGitSyncStateParams{
		ID:            result.Service.ID,
		GitLastCommit: optionalImportedString(commit),
		GitSyncedAt:   pgtypeTz(time.Now()),
		GitSyncStatus: syncStatusOK,
		GitSyncError:  nil,
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	model := fromDB(svc, isAdmin)
	result.Service = &model
	return result, nil
}

func (s *ImportService) markGitSyncFailure(ctx context.Context, id int64, message string) error {
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "git synchronization failed"
	}

	_, err := s.q.SetGitSyncState(ctx, db.SetGitSyncStateParams{
		ID:            id,
		GitLastCommit: nil,
		GitSyncedAt:   pgtypeTz(time.Now()),
		GitSyncStatus: syncStatusFailed,
		GitSyncError:  &msg,
	})
	return err
}

func (s *ImportService) markGitSyncFailureAndLog(ctx context.Context, id int64, message string) {
	if err := s.markGitSyncFailure(ctx, id, message); err != nil {
		slog.Warn("failed to persist git sync failure state", "service_id", id, "error", err)
	}
}
