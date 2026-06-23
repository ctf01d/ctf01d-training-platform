package services

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/service/avatarnorm"
)

type ServiceModel struct {
	ID                  int64
	Name                string
	PublicDescription   *string
	PrivateDescription  *string
	Author              *string
	Copyright           *string
	AvatarUrl           *string
	Public              bool
	ServiceArchiveUrl   *string
	CheckerArchiveUrl   *string
	WriteupUrl          *string
	ExploitsUrl         *string
	CheckStatus         string
	CheckedAt           *time.Time
	ServiceLocalPath    *string
	ServiceLocalSize    *int32
	ServiceLocalSha256  *string
	ServiceDownloadedAt *time.Time
	CheckerLocalPath    *string
	CheckerLocalSize    *int32
	CheckerLocalSha256  *string
	CheckerDownloadedAt *time.Time
	Ctf01dTraining      json.RawMessage
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ServiceListResult struct {
	Items   []ServiceModel
	Page    int
	PerPage int
	Total   int64
}

type CreateParams struct {
	Name               string
	PublicDescription  *string
	PrivateDescription *string
	Author             *string
	Copyright          *string
	AvatarUrl          *string
	Public             bool
	ServiceArchiveUrl  *string
	CheckerArchiveUrl  *string
	WriteupUrl         *string
	ExploitsUrl        *string
	Ctf01dTraining     json.RawMessage
}

type UpdateParams struct {
	Name               *string
	PublicDescription  *string
	PrivateDescription *string
	Author             *string
	Copyright          *string
	AvatarUrl          *string
	Public             *bool
	ServiceArchiveUrl  *string
	CheckerArchiveUrl  *string
	WriteupUrl         *string
	ExploitsUrl        *string
	Ctf01dTraining     json.RawMessage
}

type Querier interface {
	CreateService(ctx context.Context, arg db.CreateServiceParams) (db.Service, error)
	GetServiceByID(ctx context.Context, id int64) (db.Service, error)
	ListServices(ctx context.Context, arg db.ListServicesParams) ([]db.Service, error)
	CountServices(ctx context.Context, arg db.CountServicesParams) (int64, error)
	UpdateService(ctx context.Context, arg db.UpdateServiceParams) (db.Service, error)
	DeleteService(ctx context.Context, id int64) error
	SetPublic(ctx context.Context, arg db.SetPublicParams) (db.Service, error)
}

type Service struct {
	q Querier
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

func (s *Service) Create(ctx context.Context, params CreateParams, isAdmin bool) (*ServiceModel, error) {
	if err := validateServiceURLs(params.AvatarUrl, params.ServiceArchiveUrl, params.CheckerArchiveUrl, params.WriteupUrl, params.ExploitsUrl); err != nil {
		return nil, err
	}

	avatarURL, err := avatarnorm.Normalize(params.AvatarUrl, "avatar_url")
	if err != nil {
		return nil, err
	}

	training := params.Ctf01dTraining
	if training == nil {
		training = json.RawMessage("{}")
	}

	dbSvc, err := s.q.CreateService(ctx, db.CreateServiceParams{
		Name:               params.Name,
		PublicDescription:  params.PublicDescription,
		PrivateDescription: params.PrivateDescription,
		Author:             params.Author,
		Copyright:          params.Copyright,
		AvatarUrl:          avatarURL,
		Public:             params.Public,
		ServiceArchiveUrl:  params.ServiceArchiveUrl,
		CheckerArchiveUrl:  params.CheckerArchiveUrl,
		WriteupUrl:         params.WriteupUrl,
		ExploitsUrl:        params.ExploitsUrl,
		CheckStatus:        checkStatusUnknown,
		Ctf01dTraining:     training,
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	svc := fromDB(dbSvc, isAdmin)
	return &svc, nil
}

func (s *Service) GetByID(ctx context.Context, id int64, isAdmin bool) (*ServiceModel, error) {
	dbSvc, err := s.q.GetServiceByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	svc := fromDB(dbSvc, isAdmin)
	return &svc, nil
}

func (s *Service) List(ctx context.Context, page, perPage int, publicFilter *bool, query *string, isAdmin bool) (*ServiceListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset, err := int32FromInt64(int64(page-1) * int64(perPage))
	if err != nil {
		return nil, err
	}
	limit, err := int32FromInt64(int64(perPage))
	if err != nil {
		return nil, err
	}

	items, err := s.q.ListServices(ctx, db.ListServicesParams{
		PublicFilter: publicFilter,
		SearchQuery:  query,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, err
	}

	total, err := s.q.CountServices(ctx, db.CountServicesParams{
		PublicFilter: publicFilter,
		SearchQuery:  query,
	})
	if err != nil {
		return nil, err
	}

	result := &ServiceListResult{
		Items:   make([]ServiceModel, len(items)),
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}
	for i, item := range items {
		result.Items[i] = fromDB(item, isAdmin)
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, id int64, params UpdateParams, isAdmin bool) (*ServiceModel, error) {
	current, err := s.q.GetServiceByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}

	avatarUrl := current.AvatarUrl
	if params.AvatarUrl != nil {
		avatarUrl = params.AvatarUrl
	}
	serviceArchiveUrl := current.ServiceArchiveUrl
	if params.ServiceArchiveUrl != nil {
		serviceArchiveUrl = params.ServiceArchiveUrl
	}
	checkerArchiveUrl := current.CheckerArchiveUrl
	if params.CheckerArchiveUrl != nil {
		checkerArchiveUrl = params.CheckerArchiveUrl
	}
	writeupUrl := current.WriteupUrl
	if params.WriteupUrl != nil {
		writeupUrl = params.WriteupUrl
	}
	exploitsUrl := current.ExploitsUrl
	if params.ExploitsUrl != nil {
		exploitsUrl = params.ExploitsUrl
	}

	if err := validateServiceURLs(avatarUrl, serviceArchiveUrl, checkerArchiveUrl, writeupUrl, exploitsUrl); err != nil {
		return nil, err
	}

	// Only the explicitly-supplied avatar is normalized; a nil avatar leaves the
	// stored value untouched (UpdateService COALESCEs nil onto the current row).
	var normalizedAvatar *string
	if params.AvatarUrl != nil {
		normalizedAvatar, err = avatarnorm.Normalize(params.AvatarUrl, "avatar_url")
		if err != nil {
			return nil, err
		}
	}

	name := current.Name
	if params.Name != nil {
		if *params.Name == "" {
			return nil, errs.NewValidationError(map[string]string{fieldName: "must be non-empty"})
		}
		name = *params.Name
	}
	public := current.Public
	if params.Public != nil {
		public = *params.Public
	}
	training := current.Ctf01dTraining
	if params.Ctf01dTraining != nil {
		training = params.Ctf01dTraining
	}

	dbSvc, err := s.q.UpdateService(ctx, db.UpdateServiceParams{
		ID:                 id,
		Name:               name,
		PublicDescription:  params.PublicDescription,
		PrivateDescription: params.PrivateDescription,
		Author:             params.Author,
		Copyright:          params.Copyright,
		AvatarUrl:          normalizedAvatar,
		Public:             public,
		ServiceArchiveUrl:  params.ServiceArchiveUrl,
		CheckerArchiveUrl:  params.CheckerArchiveUrl,
		WriteupUrl:         params.WriteupUrl,
		ExploitsUrl:        params.ExploitsUrl,
		Ctf01dTraining:     training,
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	svc := fromDB(dbSvc, isAdmin)
	return &svc, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.q.DeleteService(ctx, id)
}

func (s *Service) TogglePublic(ctx context.Context, id int64, isAdmin bool) (*ServiceModel, error) {
	current, err := s.q.GetServiceByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}

	dbSvc, err := s.q.SetPublic(ctx, db.SetPublicParams{
		ID:     id,
		Public: !current.Public,
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	svc := fromDB(dbSvc, isAdmin)
	return &svc, nil
}

func fromDB(s db.Service, isAdmin bool) ServiceModel {
	m := ServiceModel{
		ID:                s.ID,
		Name:              s.Name,
		PublicDescription: s.PublicDescription,
		Author:            s.Author,
		Copyright:         s.Copyright,
		AvatarUrl:         s.AvatarUrl,
		Public:            s.Public,
		ServiceArchiveUrl: s.ServiceArchiveUrl,
		CheckerArchiveUrl: s.CheckerArchiveUrl,
		WriteupUrl:        s.WriteupUrl,
		ExploitsUrl:       s.ExploitsUrl,
		CheckStatus:       s.CheckStatus,
		Ctf01dTraining:    s.Ctf01dTraining,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}

	if s.PrivateDescription != nil {
		m.PrivateDescription = s.PrivateDescription
	}
	if s.CheckedAt.Valid {
		m.CheckedAt = &s.CheckedAt.Time
	}
	if s.ServiceLocalPath != nil {
		m.ServiceLocalPath = s.ServiceLocalPath
	}
	if s.ServiceLocalSize != nil {
		m.ServiceLocalSize = s.ServiceLocalSize
	}
	if s.ServiceLocalSha256 != nil {
		m.ServiceLocalSha256 = s.ServiceLocalSha256
	}
	if s.ServiceDownloadedAt.Valid {
		m.ServiceDownloadedAt = &s.ServiceDownloadedAt.Time
	}
	if s.CheckerLocalPath != nil {
		m.CheckerLocalPath = s.CheckerLocalPath
	}
	if s.CheckerLocalSize != nil {
		m.CheckerLocalSize = s.CheckerLocalSize
	}
	if s.CheckerLocalSha256 != nil {
		m.CheckerLocalSha256 = s.CheckerLocalSha256
	}
	if s.CheckerDownloadedAt.Valid {
		m.CheckerDownloadedAt = &s.CheckerDownloadedAt.Time
	}

	if !isAdmin {
		m.PrivateDescription = nil
		m.ServiceLocalPath = nil
		m.ServiceLocalSize = nil
		m.ServiceLocalSha256 = nil
		m.ServiceDownloadedAt = nil
		m.CheckerLocalPath = nil
		m.CheckerLocalSize = nil
		m.CheckerLocalSha256 = nil
		m.CheckerDownloadedAt = nil
	}

	return m
}

func validHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func validAvatarURL(s string) bool {
	if strings.HasPrefix(s, "data:image/") {
		rest := s[len("data:image/"):]
		i := strings.Index(rest, ";")
		if i < 0 {
			return false
		}
		mime := rest[:i]
		// Only formats imageutil can decode (see imageutil/avatar.go); webp has
		// no registered decoder, so accepting it here would defer a confusing
		// failure to normalization.
		for _, allowed := range []string{"png", "jpeg", "gif"} {
			if mime == allowed {
				return strings.HasPrefix(rest[i+1:], "base64,")
			}
		}
		return false
	}
	return validHTTPURL(s)
}

func validateServiceURLs(avatarUrl, serviceArchiveUrl, checkerArchiveUrl, writeupUrl, exploitsUrl *string) error {
	fields := make(map[string]string)

	if avatarUrl != nil && *avatarUrl != "" && !validAvatarURL(*avatarUrl) {
		fields["avatar_url"] = "must be a valid http(s) URL or data:image URI"
	}

	urlFields := map[string]*string{
		"service_archive_url": serviceArchiveUrl,
		"checker_archive_url": checkerArchiveUrl,
		"writeup_url":         writeupUrl,
		"exploits_url":        exploitsUrl,
	}
	for field, val := range urlFields {
		if val != nil && *val != "" && !validHTTPURL(*val) {
			fields[field] = "must be a valid http(s) URL"
		}
	}

	if len(fields) > 0 {
		return errs.NewValidationError(fields)
	}
	return nil
}

func mapNotFound(err error) error {
	if repository.IsNoRows(err) {
		return errs.ErrNotFound
	}
	return err
}

func mapDBError(err error) error {
	if repository.IsDuplicateKey(err) {
		return errs.ErrConflict
	}
	return err
}

func int32FromInt64(v int64) (int32, error) {
	if v < 0 || v > maxInt32Value {
		return 0, errs.NewValidationError(map[string]string{"pagination": "offset must fit int32"})
	}
	return int32(v), nil
}

func isDuplicateKey(err error) bool {
	return repository.IsDuplicateKey(err)
}

func pgtypeTz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
