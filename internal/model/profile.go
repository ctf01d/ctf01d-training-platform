package model

import (
	"time"

	"ctf01d/internal/httpserver"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type Profile struct {
	Id              openapi_types.UUID `db:"id"         json:"id"`
	CurrentTeamName string             `db:"name"       json:"name"`
	CreatedAt       time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `db:"updated_at" json:"updated_at"`
	Role            string             `db:"role"       json:"role"`
}

type ProfileTeams struct {
	JoinedAt time.Time  `db:"joined_at" json:"joined_at"`
	LeftAt   *time.Time `db:"left_at"   json:"left_at"`
	Role     string     `db:"role"      json:"role"`
	Name     string     `db:"name"      json:"name"`
}

type ProfileWithHistory struct {
	Profile Profile
	History []ProfileTeams
}

func (p *ProfileWithHistory) ToResponse() *httpserver.ProfileResponse {
	return &httpserver.ProfileResponse{
		Id:          p.Profile.Id,
		CreatedAt:   p.Profile.CreatedAt,
		UpdatedAt:   &p.Profile.UpdatedAt,
		TeamName:    p.Profile.CurrentTeamName,
		TeamRole:    httpserver.ProfileResponseTeamRole(p.Profile.Role),
		TeamHistory: makeTeamHistory(p.History),
	}
}

func makeTeamHistory(tms []ProfileTeams) *[]httpserver.TeamHistory {
	out := []httpserver.TeamHistory{}
	for _, tm := range tms {
		out = append(out, httpserver.TeamHistory{
			Join: tm.JoinedAt,
			Left: tm.LeftAt,
			Name: tm.Name,
			Role: httpserver.TeamHistoryRole(tm.Role),
		})
	}
	return &out
}
