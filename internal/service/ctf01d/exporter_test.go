package ctf01d

import (
	"archive/zip"
	"bytes"
	"net"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

func createTestBundleZip(t *testing.T, hasChecker bool) string {
	t.Helper()
	tmpDir := t.TempDir()
	bundlePath := path.Join(tmpDir, "bundle.zip")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	if _, err := w.Create("service/README.md"); err != nil {
		t.Fatalf("create README entry: %v", err)
	}
	if hasChecker {
		fw, err := w.Create("checker/checker.py")
		if err != nil {
			t.Fatalf("create checker entry: %v", err)
		}
		if _, err := fw.Write([]byte("#!/usr/bin/env python3\nprint('checker')\n")); err != nil {
			t.Fatalf("write checker entry: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	if err := os.WriteFile(bundlePath, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write bundle zip: %v", err)
	}
	return bundlePath
}

func makeTestGame() GameParams {
	return GameParams{
		ID:              "testgame",
		Name:            "Test Game",
		StartUTC:        time.Date(2025, 10, 1, 9, 0, 0, 0, time.UTC),
		EndUTC:          time.Date(2025, 10, 1, 19, 0, 0, 0, time.UTC),
		FlagTTLMin:      1,
		BasicAttackCost: 1,
		DefenceCost:     1.0,
	}
}

func makeTestScoreboard() ScoreboardParams {
	return ScoreboardParams{
		Port:       8080,
		HtmlFolder: "./html",
		Random:     false,
	}
}

func makeTestTeams() []TeamParams {
	return []TeamParams{
		{
			ID:        "t01",
			Name:      "Team #1",
			Active:    true,
			IPAddress: "10.0.1.1",
		},
		{
			ID:        "t02",
			Name:      "Team #2",
			Active:    true,
			IPAddress: "10.0.2.1",
		},
	}
}

func makeTestCheckers(t *testing.T) []CheckerParams {
	t.Helper()
	bundlePath := createTestBundleZip(t, true)
	return []CheckerParams{
		{
			ID:                "service1",
			Name:              "Service1",
			Enabled:           true,
			ScriptWait:        10,
			RoundSleep:        30,
			ScriptRel:         "./checker.py",
			BundlePath:        bundlePath,
			CheckerFromBundle: true,
		},
	}
}

func listZipNames(t *testing.T, data []byte) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}

func hasPrefix(names []string, suffix string) bool {
	for _, n := range names {
		if strings.HasSuffix(n, suffix) {
			return true
		}
	}
	return false
}

func TestExport_BasicSuccess(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	checkers := makeTestCheckers(t)
	options := Options{
		Prefix:         "ctf01d_testgame",
		IncludeHTML:    false,
		IncludeCompose: true,
		ComposeProject: "testgame",
	}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if result.Filename != "ctf01d_testgame.zip" {
		t.Errorf("expected filename ctf01d_testgame.zip, got %s", result.Filename)
	}
	if len(result.Data) == 0 {
		t.Error("expected non-empty zip data")
	}

	names := listZipNames(t, result.Data)

	if !hasPrefix(names, "/data/config.yml") {
		t.Errorf("expected data/config.yml in zip, got: %v", names)
	}
	if !hasPrefix(names, "/data/checker_service1/checker.py") {
		t.Errorf("expected data/checker_service1/checker.py in zip, got: %v", names)
	}
	if !hasPrefix(names, "/docker-compose.yml") {
		t.Errorf("expected docker-compose.yml in zip, got: %v", names)
	}
}

func TestExport_WithHTML(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	checkers := makeTestCheckers(t)
	options := Options{
		Prefix:      "ctf01d_html",
		IncludeHTML: true,
	}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	names := listZipNames(t, result.Data)
	if !hasPrefix(names, "/data/html/") {
		t.Errorf("expected data/html/ directory in zip when IncludeHTML is true, got: %v", names)
	}
}

func TestExport_ConfigYAMLIsValid(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	checkers := makeTestCheckers(t)
	options := Options{Prefix: "ctf01d_yaml_test", IncludeHTML: false}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	configContent := extractFileFromZip(t, result.Data, "data/config.yml")
	if configContent == nil {
		t.Fatal("data/config.yml not found in zip")
	}

	if !bytes.Contains(configContent, []byte("## Combined config for ctf01d")) {
		t.Error("config.yml missing header comment")
	}
	if !bytes.Contains(configContent, []byte("testgame")) {
		t.Error("config.yml missing game id")
	}
	if !bytes.Contains(configContent, []byte("10.0.1.1")) {
		t.Error("config.yml missing team ip_address")
	}
	if !bytes.Contains(configContent, []byte("service1")) {
		t.Error("config.yml missing checker id")
	}
	if !bytes.Contains(configContent, []byte("2025-10-01 09:00:00")) {
		t.Error("config.yml missing game start time")
	}
}

func TestExport_CoffeeBreak(t *testing.T) {
	game := makeTestGame()
	start := time.Date(2025, 10, 1, 12, 0, 0, 0, time.UTC)
	end := time.Date(2025, 10, 1, 13, 0, 0, 0, time.UTC)
	game.CoffeeBreakStartUTC = &start
	game.CoffeeBreakEndUTC = &end

	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	checkers := makeTestCheckers(t)
	options := Options{Prefix: "ctf01d_coffee", IncludeHTML: false}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	configContent := extractFileFromZip(t, result.Data, "data/config.yml")
	if !bytes.Contains(configContent, []byte("coffee_break_start")) {
		t.Error("config.yml missing coffee_break_start")
	}
	if !bytes.Contains(configContent, []byte("coffee_break_end")) {
		t.Error("config.yml missing coffee_break_end")
	}
}

func TestExport_ComposeYML(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	checkers := makeTestCheckers(t)
	options := Options{
		Prefix:         "ctf01d_compose",
		IncludeHTML:    false,
		IncludeCompose: true,
		ComposeProject: "mygame",
	}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	compose := extractFileFromZip(t, result.Data, "docker-compose.yml")
	if compose == nil {
		t.Fatal("docker-compose.yml not found")
	}
	if !bytes.Contains(compose, []byte("ctf01d_jury_mygame")) {
		t.Error("docker-compose.yml missing container name")
	}
	if !bytes.Contains(compose, []byte("8080:8080")) {
		t.Error("docker-compose.yml missing port mapping")
	}
}

func TestExport_WarningsFile(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	checkers := makeTestCheckers(t)
	options := Options{
		Prefix:   "ctf01d_warn",
		Warnings: []string{"team without ip", "service missing archive"},
	}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	warnings := extractFileFromZip(t, result.Data, "EXPORT_WARNINGS.txt")
	if warnings == nil {
		t.Fatal("EXPORT_WARNINGS.txt not found")
	}
	if !bytes.Contains(warnings, []byte("team without ip")) {
		t.Error("warnings file missing expected content")
	}
}

func TestExport_TeamLogosGenerated(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	checkers := makeTestCheckers(t)
	options := Options{Prefix: "ctf01d_logos", IncludeHTML: false}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	names := listZipNames(t, result.Data)
	var logoFound bool
	for _, n := range names {
		if strings.Contains(n, "/teams/") &&
			(strings.HasSuffix(n, ".svg") || strings.HasSuffix(n, ".png")) {
			logoFound = true
		}
	}
	if !logoFound {
		t.Errorf("expected team logo files in zip, got: %v", names)
	}
}

func TestExport_ServiceArchives(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	bundlePath := createTestBundleZip(t, true)
	checkers := []CheckerParams{
		{
			ID:                "svc1",
			Name:              "Svc1",
			Enabled:           true,
			ScriptWait:        10,
			RoundSleep:        30,
			BundlePath:        bundlePath,
			CheckerFromBundle: true,
		},
	}
	options := Options{Prefix: "ctf01d_arch", IncludeHTML: false}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	archive := extractFileFromZip(t, result.Data, "archives/services/svc1.zip")
	if archive == nil {
		t.Error("expected archives/services/svc1.zip in zip")
	}
}

func TestExport_CheckerFromBundleWithoutChecker(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	bundlePath := createTestBundleZip(t, false)
	checkers := []CheckerParams{
		{
			ID:                "svc_no_checker",
			Name:              "SvcNoChecker",
			Enabled:           true,
			ScriptWait:        10,
			RoundSleep:        30,
			BundlePath:        bundlePath,
			CheckerFromBundle: true,
		},
	}
	options := Options{Prefix: "ctf01d_nocheck", IncludeHTML: false}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	dummy := extractFileFromZip(t, result.Data, "data/checker_svc_no_checker/checker.py")
	if dummy == nil {
		t.Error("expected dummy checker.py when bundle has no checker dir")
	}
}

func TestExport_ExtraTeamFields(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := []TeamParams{
		{
			ID:        "t01",
			Name:      "Team Extra",
			Active:    true,
			IPAddress: "10.0.1.1",
			Ctf01dExtra: map[string]string{
				"ctf01d_type":   "attack",
				"ctf01d_active": "false",
			},
		},
		{
			ID:        "t02",
			Name:      "Team Extra2",
			Active:    true,
			IPAddress: "10.0.2.1",
		},
	}
	checkers := makeTestCheckers(t)
	options := Options{Prefix: "ctf01d_extra", IncludeHTML: false}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	configContent := extractFileFromZip(t, result.Data, "data/config.yml")
	if !bytes.Contains(configContent, []byte("type: attack")) {
		t.Error("config.yml missing ctf01d_extra 'type' field (stripped ctf01d_ prefix)")
	}
}

func TestValidateInputs_GameIDEmpty(t *testing.T) {
	game := makeTestGame()
	game.ID = ""
	err := validateInputs(game, makeTestScoreboard(), makeTestTeams(), makeTestCheckers(t))
	if err == nil {
		t.Fatal("expected error for empty game.id")
	}
	if !strings.Contains(err.Error(), "game.id") {
		t.Errorf("error should mention game.id, got: %v", err)
	}
}

func TestValidateInputs_GameIDInvalidChars(t *testing.T) {
	game := makeTestGame()
	game.ID = "My-Game!"
	err := validateInputs(game, makeTestScoreboard(), makeTestTeams(), makeTestCheckers(t))
	if err == nil {
		t.Fatal("expected error for invalid game.id")
	}
}

func TestValidateInputs_EndBeforeStart(t *testing.T) {
	game := makeTestGame()
	game.StartUTC = time.Date(2025, 10, 1, 19, 0, 0, 0, time.UTC)
	game.EndUTC = time.Date(2025, 10, 1, 9, 0, 0, 0, time.UTC)
	err := validateInputs(game, makeTestScoreboard(), makeTestTeams(), makeTestCheckers(t))
	if err == nil {
		t.Fatal("expected error for end_utc before start_utc")
	}
}

func TestValidateInputs_FlagTTLMinOutOfRange(t *testing.T) {
	game := makeTestGame()
	game.FlagTTLMin = 0
	err := validateInputs(game, makeTestScoreboard(), makeTestTeams(), makeTestCheckers(t))
	if err == nil {
		t.Fatal("expected error for flag_ttl_min=0")
	}
}

func TestValidateInputs_BasicAttackCostOutOfRange(t *testing.T) {
	game := makeTestGame()
	game.BasicAttackCost = 501
	err := validateInputs(game, makeTestScoreboard(), makeTestTeams(), makeTestCheckers(t))
	if err == nil {
		t.Fatal("expected error for basic_attack_cost=501")
	}
}

func TestValidateInputs_PortOutOfRange(t *testing.T) {
	sb := makeTestScoreboard()
	sb.Port = 10
	err := validateInputs(makeTestGame(), sb, makeTestTeams(), makeTestCheckers(t))
	if err == nil {
		t.Fatal("expected error for port=10")
	}
}

func TestValidateInputs_NoTeams(t *testing.T) {
	err := validateInputs(makeTestGame(), makeTestScoreboard(), nil, makeTestCheckers(t))
	if err == nil {
		t.Fatal("expected error for no teams")
	}
}

func TestValidateInputs_DuplicateTeamID(t *testing.T) {
	teams := []TeamParams{
		{ID: "t01", Name: "T1", Active: true, IPAddress: "10.0.1.1"},
		{ID: "t01", Name: "T2", Active: true, IPAddress: "10.0.2.1"},
	}
	err := validateInputs(makeTestGame(), makeTestScoreboard(), teams, makeTestCheckers(t))
	if err == nil {
		t.Fatal("expected error for duplicate team id")
	}
}

func TestValidateInputs_DuplicateIP(t *testing.T) {
	teams := []TeamParams{
		{ID: "t01", Name: "T1", Active: true, IPAddress: "10.0.1.1"},
		{ID: "t02", Name: "T2", Active: true, IPAddress: "10.0.1.1"},
	}
	err := validateInputs(makeTestGame(), makeTestScoreboard(), teams, makeTestCheckers(t))
	if err == nil {
		t.Fatal("expected error for duplicate ip")
	}
}

func TestValidateInputs_NoCheckers(t *testing.T) {
	err := validateInputs(makeTestGame(), makeTestScoreboard(), makeTestTeams(), nil)
	if err == nil {
		t.Fatal("expected error for no checkers")
	}
}

func TestValidateInputs_ScriptWaitTooLow(t *testing.T) {
	checkers := []CheckerParams{
		{ID: "svc1", Name: "S1", Enabled: true, ScriptWait: 3, RoundSleep: 30, ScriptRel: "./checker.py"},
	}
	err := validateInputs(makeTestGame(), makeTestScoreboard(), makeTestTeams(), checkers)
	if err == nil {
		t.Fatal("expected error for script_wait < 5")
	}
}

func TestValidateInputs_RoundSleepTooLow(t *testing.T) {
	checkers := []CheckerParams{
		{ID: "svc1", Name: "S1", Enabled: true, ScriptWait: 10, RoundSleep: 20, ScriptRel: "./checker.py"},
	}
	err := validateInputs(makeTestGame(), makeTestScoreboard(), makeTestTeams(), checkers)
	if err == nil {
		t.Fatal("expected error for round_sleep < script_wait * 3")
	}
}

func TestValidateInputs_EmptyScriptRel(t *testing.T) {
	checkers := []CheckerParams{
		{ID: "svc1", Name: "S1", Enabled: true, ScriptWait: 10, RoundSleep: 30, ScriptRel: ""},
	}
	err := validateInputs(makeTestGame(), makeTestScoreboard(), makeTestTeams(), checkers)
	if err == nil {
		t.Fatal("expected error for empty script_rel")
	}
}

func TestNormalizeID(t *testing.T) {
	tests := []struct{ in, want string }{
		{"MyService", "myservice"},
		{"My-Service!", "my_service"},
		{"___test___", "test"},
		{"ABC 123", "abc_123"},
	}
	for _, tt := range tests {
		got := normalizeID(tt.in)
		if got != tt.want {
			t.Errorf("normalizeID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestHydrateCheckers_Defaults(t *testing.T) {
	checkers := []CheckerParams{
		{ID: "s1", ScriptWait: 0, RoundSleep: 0, BundlePath: "/tmp/bundle.zip", CheckerFromBundle: true},
	}
	hydrateCheckers(checkers)
	if checkers[0].ScriptWait != 10 {
		t.Errorf("expected ScriptWait=10, got %d", checkers[0].ScriptWait)
	}
	if checkers[0].RoundSleep != 30 {
		t.Errorf("expected RoundSleep=30, got %d", checkers[0].RoundSleep)
	}
}

func TestExport_NoBundleFiles(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	checkers := []CheckerParams{
		{
			ID:         "svc_files",
			Name:       "SvcFiles",
			Enabled:    true,
			ScriptWait: 10,
			RoundSleep: 30,
			ScriptRel:  "./checker.py",
			Files: []CheckerFile{
				{Src: "", Rel: "checker.py"},
			},
		},
	}
	options := Options{Prefix: "ctf01d_files", IncludeHTML: false}

	result, err := Export(game, scoreboard, teams, checkers, options)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	content := extractFileFromZip(t, result.Data, "data/checker_svc_files/checker.py")
	if content == nil {
		t.Error("expected checker.py for file-based checker")
	}
	if !bytes.Contains(content, []byte("dummy checker")) {
		t.Error("expected dummy checker content")
	}
}

func TestBuildYAMLConfig_DeterministicOrder(t *testing.T) {
	game := makeTestGame()
	scoreboard := makeTestScoreboard()
	teams := makeTestTeams()
	checkers := makeTestCheckers(t)

	yaml1, err := buildYAMLConfig(game, scoreboard, teams, checkers)
	if err != nil {
		t.Fatalf("buildYAMLConfig: %v", err)
	}
	yaml2, err := buildYAMLConfig(game, scoreboard, teams, checkers)
	if err != nil {
		t.Fatalf("buildYAMLConfig: %v", err)
	}
	if yaml1 != yaml2 {
		t.Error("YAML config should be deterministic")
	}
}

func TestExtFromMIME(t *testing.T) {
	tests := []struct{ mime, ext string }{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"image/svg+xml", ".svg"},
		{"image/gif", ".gif"},
		{"unknown", ".png"},
	}
	for _, tt := range tests {
		got := extFromMIME(tt.mime)
		if got != tt.ext {
			t.Errorf("extFromMIME(%q) = %q, want %q", tt.mime, got, tt.ext)
		}
	}
}

func extractFileFromZip(t *testing.T, data []byte, suffix string) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, suffix) {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open %s: %v", f.Name, err)
			}
			defer rc.Close()
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(rc); err != nil {
				t.Fatalf("read %s: %v", f.Name, err)
			}
			return buf.Bytes()
		}
	}
	return nil
}

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip string
		ok bool
	}{
		{"127.0.0.1", false},
		{"::1", false},
		{"10.0.0.1", false},
		{"192.168.1.1", false},
		{"172.16.0.1", false},
		{"169.254.169.254", false},
		{"0.0.0.1", false},
		{"fd00::1", false},
		{"fc00::1", false},
		{"fe80::1", false},
		{"::", false},
		{"100.64.0.1", false},
		{"100.127.255.254", false},
		{"198.18.0.1", false},
		{"198.19.255.254", false},
		{"192.0.2.1", false},
		{"198.51.100.1", false},
		{"203.0.113.1", false},
		{"2001:db8::1", false},
		{"224.0.0.1", false},
		{"239.255.255.255", false},
		{"ff02::1", false},
		{"240.0.0.1", false},
		{"255.255.255.255", false},
		{"192.0.0.1", false},
		{"192.88.99.1", false},
		{"64:ff9b:1::1", false},
		{"100::1", false},
		{"100:0:0:1::1", false},
		{"2001:2::1", false},
		{"2002:c0a8:101::", false},
		{"3fff::1", false},
		{"5f00::1", false},
		{"93.184.216.34", true},
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"2606:4700:4700::1111", true},
		{"2001:4860:4860::8888", true},
	}
	for _, tc := range cases {
		blocked := isBlockedIP(net.ParseIP(tc.ip))
		if tc.ok && blocked {
			t.Errorf("isBlockedIP(%q): expected allowed, got blocked", tc.ip)
		}
		if !tc.ok && !blocked {
			t.Errorf("isBlockedIP(%q): expected blocked, got allowed", tc.ip)
		}
	}
}
