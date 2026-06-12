package ctf01d

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"gopkg.in/yaml.v3"

	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

func TestFullExport_WithBuilder(t *testing.T) {
	mq := &mockBuilderQuerier{
		game: db.Game{
			ID:       42,
			Name:     strPtr("CTF 2025"),
			StartsAt: pgtype.Timestamptz{Time: time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC), Valid: true},
			EndsAt:   pgtype.Timestamptz{Time: time.Date(2025, 6, 1, 18, 0, 0, 0, time.UTC), Valid: true},
		},
		gameTeams: []db.GameTeam{
			{ID: 1, GameID: 42, TeamID: 10, IpAddress: strPtr("10.10.0.1"), Ctf01dID: strPtr("alpha")},
			{ID: 2, GameID: 42, TeamID: 20, IpAddress: strPtr("10.10.0.2"), Ctf01dID: strPtr("beta")},
		},
		serviceIDs: []int64{100},
		services: map[int64]db.Service{
			100: {
				ID:               100,
				Name:             "web-task",
				CheckerLocalPath: strPtr(path.Join("testdata", "service_bundle.zip")),
				ServiceLocalPath: strPtr(path.Join("testdata", "service_bundle.zip")),
				Ctf01dTraining:   json.RawMessage(`{"script_wait": 10, "round_sleep": 30}`),
			},
		},
		teams: map[int64]db.Team{
			10: {ID: 10, Name: "Alpha Team"},
			20: {ID: 20, Name: "Beta Team"},
		},
	}

	b := NewBuilder(mq)
	prefix := "ctf01d_test"
	includeHTML := true
	includeCompose := true
	req := Ctf01dExportRequest{
		Prefix:         &prefix,
		IncludeHtml:    &includeHTML,
		IncludeCompose: &includeCompose,
	}

	result, err := b.BuildParams(context.Background(), 42, req)
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}

	result.Options.Warnings = result.Warnings

	exportResult, err := Export(result.Game, result.Scoreboard, result.Teams, result.Checkers, result.Options)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if exportResult.Filename != "ctf01d_test.zip" {
		t.Errorf("Filename = %q, want ctf01d_test.zip", exportResult.Filename)
	}
	if len(exportResult.Data) == 0 {
		t.Fatal("Export data is empty")
	}

	zr, err := zip.NewReader(bytes.NewReader(exportResult.Data), int64(len(exportResult.Data)))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}

	fileMap := map[string]*zip.File{}
	for _, f := range zr.File {
		fileMap[f.Name] = f
	}

	configPath := "ctf01d_test/data/config.yml"
	cf, ok := fileMap[configPath]
	if !ok {
		t.Fatalf("config.yml not found in zip; files: %v", fileMapNames(zr))
	}

	rc, err := cf.Open()
	if err != nil {
		t.Fatalf("open config.yml: %v", err)
	}
	configData, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("read config.yml: %v", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		t.Fatalf("parse config.yml: %v", err)
	}

	gameMap, ok := config["game"].(map[string]interface{})
	if !ok {
		t.Fatal("config has no game section")
	}
	if fmt.Sprintf("%v", gameMap["id"]) != "42" {
		t.Errorf("game.id = %v, want 42", gameMap["id"])
	}
	if gameMap["name"] != "CTF 2025" {
		t.Errorf("game.name = %v, want CTF 2025", gameMap["name"])
	}

	checkersSeq, ok := config["checkers"].([]interface{})
	if !ok {
		t.Fatal("config has no checkers section")
	}
	if len(checkersSeq) != 1 {
		t.Fatalf("len(checkers) = %d, want 1", len(checkersSeq))
	}
	chk := checkersSeq[0].(map[string]interface{})
	if chk["service_name"] != "web-task" {
		t.Errorf("checker name = %v, want web-task", chk["service_name"])
	}

	teamsSeq, ok := config["teams"].([]interface{})
	if !ok {
		t.Fatal("config has no teams section")
	}
	if len(teamsSeq) != 2 {
		t.Fatalf("len(teams) = %d, want 2", len(teamsSeq))
	}

	composePath := "ctf01d_test/docker-compose.yml"
	if _, ok := fileMap[composePath]; !ok {
		t.Errorf("docker-compose.yml not found; files: %v", fileMapNames(zr))
	}

	htmlPath := "ctf01d_test/data/html/"
	foundHTML := false
	for name := range fileMap {
		if len(name) > len(htmlPath) && name[:len(htmlPath)] == htmlPath {
			foundHTML = true
			break
		}
	}
	if !foundHTML {
		t.Errorf("html directory not found in zip")
	}

	checkerDir := "ctf01d_test/data/checker_web_task/"
	foundChecker := false
	for name := range fileMap {
		if len(name) > len(checkerDir) && name[:len(checkerDir)] == checkerDir {
			foundChecker = true
			break
		}
	}
	if !foundChecker {
		t.Errorf("checker directory not found; files: %v", fileMapNames(zr))
	}
}

func TestFullExport_ValidationErrors(t *testing.T) {
	mq := &mockBuilderQuerier{
		game: db.Game{
			ID:   1,
			Name: strPtr("Broken"),
		},
		gameTeams:  []db.GameTeam{},
		serviceIDs: []int64{},
		teams:      map[int64]db.Team{},
		services:   map[int64]db.Service{},
	}

	b := NewBuilder(mq)
	result, err := b.BuildParams(context.Background(), 1, Ctf01dExportRequest{})
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}

	_, exportErr := Export(result.Game, result.Scoreboard, result.Teams, result.Checkers, result.Options)
	if exportErr == nil {
		t.Fatal("expected export error for invalid data")
	}

	exportErrTyped, ok := exportErr.(*ExportError)
	if !ok {
		t.Fatalf("expected *ExportError, got %T", exportErr)
	}

	foundNoStart := false
	foundNoTeams := false
	foundNoCheckers := false
	for _, e := range exportErrTyped.Errors {
		if e == "game.start_utc is required" {
			foundNoStart = true
		}
		if e == "at least one team is required" {
			foundNoTeams = true
		}
		if e == "at least one checker is required" {
			foundNoCheckers = true
		}
	}
	if !foundNoStart {
		t.Errorf("missing 'game.start_utc is required' error; got: %v", exportErrTyped.Errors)
	}
	if !foundNoTeams {
		t.Errorf("missing 'at least one team is required' error; got: %v", exportErrTyped.Errors)
	}
	if !foundNoCheckers {
		t.Errorf("missing 'at least one checker is required' error; got: %v", exportErrTyped.Errors)
	}
}

func TestBuildParams_WithActualBundle(t *testing.T) {
	bundlePath := path.Join("testdata", "service_bundle.zip")
	if _, err := os.Stat(bundlePath); err != nil {
		t.Skip("testdata/service_bundle.zip not found")
	}

	mq := &mockBuilderQuerier{
		game: db.Game{
			ID:       1,
			Name:     strPtr("Bundle Test"),
			StartsAt: pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true},
			EndsAt:   pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 18, 0, 0, 0, time.UTC), Valid: true},
		},
		gameTeams: []db.GameTeam{
			{ID: 1, GameID: 1, TeamID: 10, IpAddress: strPtr("10.0.0.1")},
		},
		serviceIDs: []int64{100},
		services: map[int64]db.Service{
			100: {
				ID:               100,
				Name:             "bundled_service",
				CheckerLocalPath: &bundlePath,
				ServiceLocalPath: &bundlePath,
				Ctf01dTraining:   json.RawMessage(`{}`),
			},
		},
		teams: map[int64]db.Team{
			10: {ID: 10, Name: "TestTeam"},
		},
	}

	b := NewBuilder(mq)
	result, err := b.BuildParams(context.Background(), 1, Ctf01dExportRequest{})
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}

	if len(result.Checkers) != 1 {
		t.Fatalf("len(Checkers) = %d, want 1", len(result.Checkers))
	}
	if result.Checkers[0].BundlePath != bundlePath {
		t.Errorf("BundlePath = %q, want %q", result.Checkers[0].BundlePath, bundlePath)
	}
	if !result.Checkers[0].CheckerFromBundle {
		t.Error("CheckerFromBundle = false, want true")
	}

	exportResult, err := Export(result.Game, result.Scoreboard, result.Teams, result.Checkers, result.Options)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(exportResult.Data), int64(len(exportResult.Data)))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}

	foundCheckerPy := false
	for _, f := range zr.File {
		if f.Name == "ctf01d_package/data/checker_bundled_service/checker.py" {
			foundCheckerPy = true
		}
	}
	if !foundCheckerPy {
		t.Errorf("checker.py not found in exported zip; files: %v", fileMapNames(zr))
	}
}

func fileMapNames(zr *zip.Reader) []string {
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}
