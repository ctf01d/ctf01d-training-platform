package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/storage"
)

var zipMagic = []byte{0x50, 0x4B, 0x03, 0x04}

const (
	archiveDownloadTimeout = 5 * time.Minute
	maxRedirects           = 10
	zipMagicSize           = 4
	uploadLimitOverhead    = 1
)

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
			Timeout: archiveDownloadTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return errors.New("too many redirects")
				}
				host := req.URL.Hostname()
				if host == "" {
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

func ssrfSafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolving host %s: %w", host, err)
	}
	var allowed []net.IPAddr
	for _, ip := range ips {
		if blockedIPCheck(ip.IP) {
			return nil, errBlockedHost
		}
		allowed = append(allowed, ip)
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("no resolved addresses for host %s", host)
	}
	d := net.Dialer{}
	var lastErr error
	for _, ip := range allowed {
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (s *ArchiveService) Redownload(ctx context.Context, id int64, isAdmin bool) (*ServiceModel, error) {
	svc, err := s.q.GetServiceByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}

	now := time.Now()

	if svc.ServiceArchiveUrl != nil && *svc.ServiceArchiveUrl != "" {
		info, err := s.downloadAndSave(ctx, *svc.ServiceArchiveUrl, fmt.Sprintf("services/%d/service.zip", id))
		if err != nil {
			return nil, fmt.Errorf("downloading service archive: %w", err)
		}
		size, err := int32Size(info.Size)
		if err != nil {
			return nil, err
		}
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
		size, err := int32Size(info.Size)
		if err != nil {
			return nil, err
		}
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
		return nil, mapNotFound(err)
	}

	now := time.Now()

	if serviceFile != nil {
		info, err := s.saveUploaded(ctx, serviceFile, fmt.Sprintf("services/%d/service.zip", id))
		if err != nil {
			return nil, fmt.Errorf("saving service archive: %w", err)
		}
		size, err := int32Size(info.Size)
		if err != nil {
			return nil, err
		}
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
		size, err := int32Size(info.Size)
		if err != nil {
			return nil, err
		}
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
		return nil, "", mapNotFound(err)
	}

	var key string
	switch strings.ToLower(kind) {
	case kindService:
		if svc.ServiceLocalPath == nil {
			return nil, "", errs.ErrNotFound
		}
		key = *svc.ServiceLocalPath
	case kindChecker:
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

var errBlockedHost = errors.New("URL resolves to a blocked or private address")

var blockedIPCheck = isBlockedIP

var (
	blockedNets     []*net.IPNet
	blockedNetsOnce sync.Once
)

func initBlockedNets() {
	blockedNetsOnce.Do(func() {
		blockedNets = mustParseCIDRs(
			"127.0.0.0/8", "::1/128",
			"169.254.0.0/16", "fe80::/10",
			"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7",
			"0.0.0.0/8", "::/128",
			"100.64.0.0/10",
			"198.18.0.0/15",
			"192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24", "2001:db8::/32",
			"224.0.0.0/4", "ff00::/8",
			"240.0.0.0/4",
			"192.0.0.0/24",
			"192.88.99.0/24",
			"64:ff9b:1::/48",
			"100::/64",
			"100:0:0:1::/64",
			"2001:2::/48",
			"2002::/16",
			"3fff::/20",
			"5f00::/16",
		)
	})
}

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	var nets []*net.IPNet
	for _, s := range cidrs {
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			panic(err)
		}
		nets = append(nets, n)
	}
	return nets
}

func isBlockedIP(ip net.IP) bool {
	initBlockedNets()
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *ArchiveService) downloadAndSave(ctx context.Context, archiveURL, key string) (storage.FileInfo, error) {
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

	header := make([]byte, zipMagicSize)
	n, err := io.ReadFull(resp.Body, header)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("reading archive header: %w", err)
	}
	if !bytes.Equal(header[:n], zipMagic) {
		return storage.FileInfo{}, errors.New("downloaded file is not a valid ZIP archive")
	}

	limited := io.LimitReader(resp.Body, s.maxUploadBytes+uploadLimitOverhead)
	reader := io.MultiReader(bytes.NewReader(header), limited)

	info, err := s.store.Save(ctx, key, reader)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("saving to storage: %w", err)
	}

	if info.Size > s.maxUploadBytes {
		if err := s.store.Delete(ctx, key); err != nil {
			return storage.FileInfo{}, fmt.Errorf("deleting oversized archive: %w", err)
		}
		return storage.FileInfo{}, fmt.Errorf("archive exceeds maximum size (%d bytes)", s.maxUploadBytes)
	}

	return info, nil
}

func (s *ArchiveService) saveUploaded(ctx context.Context, r io.Reader, key string) (storage.FileInfo, error) {
	header := make([]byte, zipMagicSize)
	n, err := io.ReadFull(r, header)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("reading archive header: %w", err)
	}
	if !bytes.Equal(header[:n], zipMagic) {
		return storage.FileInfo{}, errs.NewValidationError(map[string]string{
			fieldArchive: "file is not a valid ZIP archive",
		})
	}

	limited := io.LimitReader(r, s.maxUploadBytes+uploadLimitOverhead)
	reader := io.MultiReader(bytes.NewReader(header), limited)

	info, err := s.store.Save(ctx, key, reader)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("saving to storage: %w", err)
	}

	if info.Size > s.maxUploadBytes {
		if err := s.store.Delete(ctx, key); err != nil {
			return storage.FileInfo{}, fmt.Errorf("deleting oversized archive: %w", err)
		}
		return storage.FileInfo{}, fmt.Errorf("archive exceeds maximum size (%d bytes)", s.maxUploadBytes)
	}

	return info, nil
}
