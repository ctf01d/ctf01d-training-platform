package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/storage"
)

var zipMagic = []byte{0x50, 0x4B, 0x03, 0x04}

type ArchiveQuerier interface {
	GetServiceByID(ctx context.Context, id int64) (db.Service, error)
	SetServiceLocal(ctx context.Context, arg db.SetServiceLocalParams) (db.Service, error)
	SetCheckerLocal(ctx context.Context, arg db.SetCheckerLocalParams) (db.Service, error)
}

type ArchiveService struct {
	q              ArchiveQuerier
	store          storage.Storage
	maxUploadBytes int64
	httpClient     *http.Client
}

func NewArchiveService(q ArchiveQuerier, store storage.Storage, maxUploadBytes int64) *ArchiveService {
	return &ArchiveService{
		q:              q,
		store:          store,
		maxUploadBytes: maxUploadBytes,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (s *ArchiveService) Redownload(ctx context.Context, id int64, isAdmin bool) (*ServiceModel, error) {
	svc, err := s.q.GetServiceByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "service")
	}

	now := time.Now()

	if svc.ServiceArchiveUrl != nil && *svc.ServiceArchiveUrl != "" {
		info, err := s.downloadAndSave(ctx, *svc.ServiceArchiveUrl, fmt.Sprintf("services/%d/service.zip", id))
		if err != nil {
			return nil, fmt.Errorf("downloading service archive: %w", err)
		}
		size := int32(info.Size)
		path := fmt.Sprintf("services/%d/service.zip", id)
		svc, err = s.q.SetServiceLocal(ctx, db.SetServiceLocalParams{
			ID:                  id,
			ServiceLocalPath:    &path,
			ServiceLocalSize:    &size,
			ServiceLocalSha256:  &info.SHA256,
			ServiceDownloadedAt: pgtypeTz(now),
		})
		if err != nil {
			return nil, mapDBError(err)
		}
	}

	if svc.CheckerArchiveUrl != nil && *svc.CheckerArchiveUrl != "" {
		info, err := s.downloadAndSave(ctx, *svc.CheckerArchiveUrl, fmt.Sprintf("services/%d/checker.zip", id))
		if err != nil {
			return nil, fmt.Errorf("downloading checker archive: %w", err)
		}
		size := int32(info.Size)
		path := fmt.Sprintf("services/%d/checker.zip", id)
		svc, err = s.q.SetCheckerLocal(ctx, db.SetCheckerLocalParams{
			ID:                  id,
			CheckerLocalPath:    &path,
			CheckerLocalSize:    &size,
			CheckerLocalSha256:  &info.SHA256,
			CheckerDownloadedAt: pgtypeTz(now),
		})
		if err != nil {
			return nil, mapDBError(err)
		}
	}

	result := fromDB(svc, isAdmin)
	return &result, nil
}

func (s *ArchiveService) UploadArchives(ctx context.Context, id int64, serviceFile, checkerFile io.Reader, isAdmin bool) (*ServiceModel, error) {
	svc, err := s.q.GetServiceByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "service")
	}

	now := time.Now()

	if serviceFile != nil {
		info, err := s.saveUploaded(ctx, serviceFile, fmt.Sprintf("services/%d/service.zip", id))
		if err != nil {
			return nil, fmt.Errorf("saving service archive: %w", err)
		}
		size := int32(info.Size)
		path := fmt.Sprintf("services/%d/service.zip", id)
		svc, err = s.q.SetServiceLocal(ctx, db.SetServiceLocalParams{
			ID:                  id,
			ServiceLocalPath:    &path,
			ServiceLocalSize:    &size,
			ServiceLocalSha256:  &info.SHA256,
			ServiceDownloadedAt: pgtypeTz(now),
		})
		if err != nil {
			return nil, mapDBError(err)
		}
	}

	if checkerFile != nil {
		info, err := s.saveUploaded(ctx, checkerFile, fmt.Sprintf("services/%d/checker.zip", id))
		if err != nil {
			return nil, fmt.Errorf("saving checker archive: %w", err)
		}
		size := int32(info.Size)
		path := fmt.Sprintf("services/%d/checker.zip", id)
		svc, err = s.q.SetCheckerLocal(ctx, db.SetCheckerLocalParams{
			ID:                  id,
			CheckerLocalPath:    &path,
			CheckerLocalSize:    &size,
			CheckerLocalSha256:  &info.SHA256,
			CheckerDownloadedAt: pgtypeTz(now),
		})
		if err != nil {
			return nil, mapDBError(err)
		}
	}

	result := fromDB(svc, isAdmin)
	return &result, nil
}

func (s *ArchiveService) OpenLocal(ctx context.Context, id int64, kind string) (io.ReadSeekCloser, string, error) {
	svc, err := s.q.GetServiceByID(ctx, id)
	if err != nil {
		return nil, "", mapNotFound(err, "service")
	}

	var key string
	switch strings.ToLower(kind) {
	case "service":
		if svc.ServiceLocalPath == nil {
			return nil, "", errs.ErrNotFound
		}
		key = *svc.ServiceLocalPath
	case "checker":
		if svc.CheckerLocalPath == nil {
			return nil, "", errs.ErrNotFound
		}
		key = *svc.CheckerLocalPath
	default:
		return nil, "", errs.NewValidationError(map[string]string{"kind": "must be 'service' or 'checker'"})
	}

	rc, err := s.store.Open(ctx, key)
	if err != nil {
		return nil, "", fmt.Errorf("opening archive: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.zip", svc.Name, kind)
	return rc, filename, nil
}

var errBlockedHost = fmt.Errorf("URL resolves to a blocked or private address")

var ssrfCheckHost = checkURLHost

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 0 {
			return true
		}
	}
	return false
}

func checkURLHost(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL has no host")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolving host: %w", err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return errBlockedHost
		}
	}
	return nil
}

func (s *ArchiveService) downloadAndSave(ctx context.Context, archiveURL, key string) (storage.FileInfo, error) {
	if err := ssrfCheckHost(archiveURL); err != nil {
		return storage.FileInfo{}, fmt.Errorf("checking URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return storage.FileInfo{}, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	header := make([]byte, 4)
	n, err := io.ReadFull(resp.Body, header)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("reading archive header: %w", err)
	}
	if !bytes.Equal(header[:n], zipMagic) {
		return storage.FileInfo{}, fmt.Errorf("downloaded file is not a valid ZIP archive")
	}

	limited := io.LimitReader(resp.Body, s.maxUploadBytes+1)
	reader := io.MultiReader(bytes.NewReader(header), limited)

	info, err := s.store.Save(ctx, key, reader)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("saving to storage: %w", err)
	}

	if info.Size > s.maxUploadBytes {
		s.store.Delete(ctx, key)
		return storage.FileInfo{}, fmt.Errorf("archive exceeds maximum size (%d bytes)", s.maxUploadBytes)
	}

	return info, nil
}

func (s *ArchiveService) saveUploaded(ctx context.Context, r io.Reader, key string) (storage.FileInfo, error) {
	header := make([]byte, 4)
	n, err := io.ReadFull(r, header)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("reading archive header: %w", err)
	}
	if !bytes.Equal(header[:n], zipMagic) {
		return storage.FileInfo{}, errs.NewValidationError(map[string]string{
			"archive": "file is not a valid ZIP archive",
		})
	}

	limited := io.LimitReader(r, s.maxUploadBytes+1)
	reader := io.MultiReader(bytes.NewReader(header), limited)

	info, err := s.store.Save(ctx, key, reader)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("saving to storage: %w", err)
	}

	if info.Size > s.maxUploadBytes {
		s.store.Delete(ctx, key)
		return storage.FileInfo{}, fmt.Errorf("archive exceeds maximum size (%d bytes)", s.maxUploadBytes)
	}

	return info, nil
}
