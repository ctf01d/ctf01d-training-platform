package handler

import (
	"encoding/json"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	svcsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/services"
)

func serviceToHTTP(s svcsvc.ServiceModel) httpserver.Service {
	result := httpserver.Service{
		Id:                s.ID,
		Name:              s.Name,
		Public:            s.Public,
		CheckStatus:       httpserver.ServiceCheckStatus(s.CheckStatus),
		PublicDescription: s.PublicDescription,
		Author:            s.Author,
		Copyright:         s.Copyright,
		AvatarUrl:         s.AvatarUrl,
		ServiceArchiveUrl: s.ServiceArchiveUrl,
		CheckerArchiveUrl: s.CheckerArchiveUrl,
		WriteupUrl:        s.WriteupUrl,
		ExploitsUrl:       s.ExploitsUrl,
		CreatedAt:         &s.CreatedAt,
		UpdatedAt:         &s.UpdatedAt,
	}

	if s.Ctf01dTraining != nil {
		var m map[string]interface{}
		if err := json.Unmarshal(s.Ctf01dTraining, &m); err == nil {
			result.Ctf01dTraining = &m
		}
	}

	if s.PrivateDescription != nil {
		result.PrivateDescription = s.PrivateDescription
	}

	if s.CheckedAt != nil {
		result.CheckedAt = s.CheckedAt
	}

	if s.ServiceLocalPath != nil {
		meta := &httpserver.ServiceArchiveMeta{
			Sha256: s.ServiceLocalSha256,
		}
		if s.ServiceLocalSize != nil {
			size := int64(*s.ServiceLocalSize)
			meta.Size = &size
		}
		if s.ServiceDownloadedAt != nil {
			meta.DownloadedAt = s.ServiceDownloadedAt
		}
		result.ServiceArchive = meta
	}

	if s.CheckerLocalPath != nil {
		meta := &httpserver.ServiceArchiveMeta{
			Sha256: s.CheckerLocalSha256,
		}
		if s.CheckerLocalSize != nil {
			size := int64(*s.CheckerLocalSize)
			meta.Size = &size
		}
		if s.CheckerDownloadedAt != nil {
			meta.DownloadedAt = s.CheckerDownloadedAt
		}
		result.CheckerArchive = meta
	}

	return result
}

func importResultToHTTP(r *svcsvc.ImportResult) httpserver.ImportResult {
	warnings := r.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return httpserver.ImportResult{
		Service:  serviceToHTTP(*r.Service),
		Warnings: warnings,
	}
}
