package model

import (
	"database/sql"

	"ctf01d/internal/helper"
	"ctf01d/internal/httpserver"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type Team struct {
	Id           openapi_types.UUID `db:"id"            json:"id"`
	Name         string             `db:"name"          json:"name"`
	Description  sql.NullString     `db:"description"   json:"description"`
	UniversityId openapi_types.UUID `db:"university_id" json:"university_id"`
	SocialLinks  sql.NullString     `db:"social_links"  json:"social_links"`
	AvatarUrl    sql.NullString     `db:"avatar_url"    json:"avatar_url"`
	University   sql.NullString
}

func (t *Team) ToResponse() *httpserver.TeamResponse {
	var avatarUrl string
	if t.AvatarUrl.Valid {
		avatarUrl = t.AvatarUrl.String
	} else {
		avatarUrl = helper.WithDefault(t.Name)
	}
	return &httpserver.TeamResponse{
		Id:          t.Id,
		Name:        t.Name,
		Description: &t.Description.String,
		University:  &t.University.String,
		SocialLinks: &t.SocialLinks.String,
		AvatarUrl:   &avatarUrl,
	}
}

func NewTeamsFromModels(ts []*Team) []*httpserver.TeamResponse {
	var teams []*httpserver.TeamResponse
	for _, t := range ts {
		teams = append(teams, t.ToResponse())
	}
	return teams
}
