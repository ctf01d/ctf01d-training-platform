package universities

import (
	"context"
	"net/url"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/service/avatarnorm"
)

type University struct {
	ID        int64     `json:"id"`
	Name      *string   `json:"name"`
	SiteUrl   *string   `json:"site_url"`
	AvatarUrl *string   `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UniversityListResult struct {
	Items   []University `json:"items"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
	Total   int64        `json:"total"`
}

type CreateParams struct {
	Name      *string `json:"name"`
	SiteUrl   *string `json:"site_url"`
	AvatarUrl *string `json:"avatar_url"`
}

type UpdateParams struct {
	Name      *string `json:"name"`
	SiteUrl   *string `json:"site_url"`
	AvatarUrl *string `json:"avatar_url"`
}

type Querier interface {
	CreateUniversity(ctx context.Context, arg db.CreateUniversityParams) (db.University, error)
	GetUniversityByID(ctx context.Context, id int64) (db.University, error)
	ListUniversities(ctx context.Context, arg db.ListUniversitiesParams) ([]db.University, error)
	CountUniversities(ctx context.Context, searchQuery *string) (int64, error)
	UpdateUniversity(ctx context.Context, arg db.UpdateUniversityParams) (db.University, error)
	DeleteUniversity(ctx context.Context, id int64) error
}

type Service struct {
	q Querier
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

func (s *Service) Create(ctx context.Context, params CreateParams) (*University, error) {
	if err := validateURLs(params.SiteUrl); err != nil {
		return nil, err
	}
	avatarURL, err := avatarnorm.Normalize(params.AvatarUrl, "avatar_url")
	if err != nil {
		return nil, err
	}
	dbUni, err := s.q.CreateUniversity(ctx, db.CreateUniversityParams{
		Name:      params.Name,
		SiteUrl:   params.SiteUrl,
		AvatarUrl: avatarURL,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	u := fromDB(dbUni)
	return &u, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*University, error) {
	dbUni, err := s.q.GetUniversityByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	u := fromDB(dbUni)
	return &u, nil
}

func (s *Service) List(ctx context.Context, page, perPage int, query *string) (*UniversityListResult, error) {
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

	items, err := s.q.ListUniversities(ctx, db.ListUniversitiesParams{
		Limit:       limit,
		Offset:      offset,
		SearchQuery: query,
	})
	if err != nil {
		return nil, err
	}

	total, err := s.q.CountUniversities(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &UniversityListResult{
		Items:   make([]University, len(items)),
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}
	for i, item := range items {
		result.Items[i] = fromDB(item)
	}

	return result, nil
}

func (s *Service) Update(ctx context.Context, id int64, params UpdateParams) (*University, error) {
	if err := validateURLs(params.SiteUrl); err != nil {
		return nil, err
	}
	avatarURL, err := avatarnorm.Normalize(params.AvatarUrl, "avatar_url")
	if err != nil {
		return nil, err
	}
	dbUni, err := s.q.UpdateUniversity(ctx, db.UpdateUniversityParams{
		ID:        id,
		Name:      params.Name,
		SiteUrl:   params.SiteUrl,
		AvatarUrl: avatarURL,
	})
	if err != nil {
		return nil, mapNotFound(err)
	}
	u := fromDB(dbUni)
	return &u, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.q.DeleteUniversity(ctx, id)
}

func fromDB(u db.University) University {
	return University{
		ID:        u.ID,
		Name:      u.Name,
		SiteUrl:   u.SiteUrl,
		AvatarUrl: u.AvatarUrl,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
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

const (
	minInt32 = -1 << 31
	maxInt32 = 1<<31 - 1
)

func int32FromInt64(v int64) (int32, error) {
	if v < minInt32 || v > maxInt32 {
		return 0, errs.NewValidationError(map[string]string{"pagination": "offset must fit int32"})
	}
	return int32(v), nil
}

func validateURLs(urls ...*string) error {
	for _, u := range urls {
		if u == nil || *u == "" {
			continue
		}
		parsed, err := url.Parse(*u)
		if err != nil {
			return errs.NewValidationError(map[string]string{"url": "invalid URL format"})
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errs.NewValidationError(map[string]string{"url": "URL must use http or https scheme"})
		}
	}
	return nil
}
