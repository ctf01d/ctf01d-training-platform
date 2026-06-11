package ctf01d

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	gameIDRe    = regexp.MustCompile(`^[a-z0-9]+$`)
	ipRe        = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)
	trailingUnd = regexp.MustCompile(`^_+|_+$`)
)

type ExportResult struct {
	Filename string
	Data     []byte
	Size     int
}

func Export(game GameParams, scoreboard ScoreboardParams, teams []TeamParams, checkers []CheckerParams, options Options) (*ExportResult, error) {
	if teams == nil {
		teams = []TeamParams{}
	}
	if checkers == nil {
		checkers = []CheckerParams{}
	}

	options = applyOptionDefaults(options)

	hydrateCheckers(checkers)
	if err := validateInputs(game, scoreboard, teams, checkers); err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "ctf01d_export_*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	root := path.Join(tmpDir, options.Prefix)
	dataDir := path.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	htmlSource := options.HtmlSourcePath
	if options.IncludeHTML {
		if htmlSource == "" || !dirExists(htmlSource) {
			htmlSource = buildFallbackHTML(tmpDir)
		} else {
			abs, err := filepath.Abs(htmlSource)
			if err != nil {
				return nil, fmt.Errorf("invalid html_source_path: %w", err)
			}
			clean := filepath.Clean(abs)
			if strings.Contains(clean, "..") {
				return nil, fmt.Errorf("invalid html_source_path: must not contain '..' components")
			}
			htmlSource = clean
		}
		copyTree(htmlSource, path.Join(dataDir, "html"))
	}

	downloadsDir := path.Join(tmpDir, "downloads")
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create downloads dir: %w", err)
	}
	ensureTeamLogos(teams, dataDir, downloadsDir)

	materializeCheckers(checkers, dataDir)
	materializeServiceArchives(checkers, root)

	cfgPath := path.Join(dataDir, "config.yml")
	cfgContent, err := buildYAMLConfig(game, scoreboard, teams, checkers)
	if err != nil {
		return nil, fmt.Errorf("build config: %w", err)
	}
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		return nil, fmt.Errorf("write config: %w", err)
	}

	if len(options.Warnings) > 0 {
		warningsPath := path.Join(root, "EXPORT_WARNINGS.txt")
		if err := os.WriteFile(warningsPath, []byte(strings.Join(options.Warnings, "\n")), 0o644); err != nil {
			return nil, fmt.Errorf("write warnings: %w", err)
		}
	}

	if options.IncludeCompose {
		composePath := path.Join(root, "docker-compose.yml")
		if err := os.WriteFile(composePath, []byte(composeYML(scoreboard, options)), 0o644); err != nil {
			return nil, fmt.Errorf("write compose: %w", err)
		}
	}

	zipData, err := packZip(root)
	if err != nil {
		return nil, fmt.Errorf("pack zip: %w", err)
	}

	return &ExportResult{
		Filename: options.Prefix + ".zip",
		Data:     zipData,
		Size:     len(zipData),
	}, nil
}

func applyOptionDefaults(o Options) Options {
	if o.Prefix == "" {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return o
		}
		o.Prefix = "ctf01d_package_" + hex.EncodeToString(b)
	}
	return o
}

func hydrateCheckers(checkers []CheckerParams) {
	for i := range checkers {
		c := &checkers[i]
		if c.BundlePath == "" {
			continue
		}
		if c.ScriptWait <= 0 {
			c.ScriptWait = 10
		}
		if c.RoundSleep < c.ScriptWait*3 {
			c.RoundSleep = c.ScriptWait * 3
		}
		if c.CheckerFromBundle && c.ScriptRel == "" {
			entrypoint := detectCheckerEntrypoint(c.BundlePath)
			if entrypoint != "" {
				c.ScriptRel = entrypoint
			} else {
				c.ScriptRel = "./checker.py"
			}
		} else if c.ScriptRel == "" {
			c.ScriptRel = "./checker.py"
		}
	}
}

func detectCheckerEntrypoint(bundlePath string) string {
	r, err := os.Open(bundlePath)
	if err != nil {
		return ""
	}
	defer r.Close()

	fi, err := r.Stat()
	if err != nil {
		return ""
	}

	zr, err := zip.NewReader(r, fi.Size())
	if err != nil {
		return ""
	}

	var relFiles []string
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		idx := findCheckerDirIndex(f.Name)
		if idx < 0 {
			continue
		}
		rel := f.Name[idx:]
		if rel == "" {
			continue
		}
		relFiles = append(relFiles, rel)
	}
	if len(relFiles) == 0 {
		return ""
	}

	candidates := []string{
		"checker.py", "checker.rb", "checker.pl", "checker.sh",
		"checker.php", "checker.go", "checker.cr", "checker.js", "checker.ts",
	}

	for _, basename := range candidates {
		chosen := pickByBasename(relFiles, basename)
		if chosen != "" {
			return "./" + chosen
		}
	}

	topLevel := filterTopLevel(relFiles)
	preferred := filterCheckerNamed(topLevel)
	var pick []string
	if len(preferred) > 0 {
		pick = preferred
	} else if len(topLevel) > 0 {
		pick = topLevel
	} else {
		pick = relFiles
	}
	if len(pick) > 0 {
		return "./" + pick[0]
	}
	return ""
}

func findCheckerDirIndex(name string) int {
	for {
		idx := strings.Index(name, "checker/")
		if idx < 0 {
			return -1
		}
		rel := name[idx+8:]
		if rel != "" {
			return idx + 8
		}
		remaining := name[idx+8:]
		if remaining == "" {
			return -1
		}
		name = remaining
	}
}

func pickByBasename(files []string, basename string) string {
	var matches []string
	for _, f := range files {
		if path.Base(f) == basename {
			matches = append(matches, f)
		}
	}
	if len(matches) == 0 {
		return ""
	}
	sort.Slice(matches, func(i, j int) bool {
		ci := strings.Count(matches[i], "/")
		cj := strings.Count(matches[j], "/")
		if ci != cj {
			return ci < cj
		}
		return len(matches[i]) < len(matches[j])
	})
	return matches[0]
}

func filterTopLevel(files []string) []string {
	var result []string
	for _, f := range files {
		if !strings.Contains(f, "/") {
			result = append(result, f)
		}
	}
	return result
}

func filterCheckerNamed(files []string) []string {
	var result []string
	for _, f := range files {
		base := path.Base(f)
		if strings.HasPrefix(base, "checker.") || base == "checker" {
			result = append(result, f)
		}
	}
	return result
}

func validateInputs(game GameParams, scoreboard ScoreboardParams, teams []TeamParams, checkers []CheckerParams) error {
	var errs []string

	gid := game.ID
	if gid == "" {
		errs = append(errs, "game.id is required")
	} else if !gameIDRe.MatchString(gid) {
		errs = append(errs, "game.id must match [a-z0-9]+")
	}

	if game.Name == "" {
		errs = append(errs, "game.name is required")
	}
	if game.StartUTC.IsZero() {
		errs = append(errs, "game.start_utc is required")
	}
	if game.EndUTC.IsZero() {
		errs = append(errs, "game.end_utc is required")
	}
	if !game.StartUTC.IsZero() && !game.EndUTC.IsZero() && !game.EndUTC.After(game.StartUTC) {
		errs = append(errs, "game.end_utc must be after game.start_utc")
	}
	if game.FlagTTLMin < 1 || game.FlagTTLMin > 25 {
		errs = append(errs, "game.flag_ttl_min must be between 1 and 25")
	}
	if game.BasicAttackCost < 1 || game.BasicAttackCost > 500 {
		errs = append(errs, "game.basic_attack_cost must be between 1 and 500")
	}

	if scoreboard.Port < 11 || scoreboard.Port > 65535 {
		errs = append(errs, "scoreboard.port must be between 11 and 65535")
	}
	if scoreboard.HtmlFolder == "" {
		errs = append(errs, "scoreboard.htmlfolder is required")
	}

	if len(teams) == 0 {
		errs = append(errs, "at least one team is required")
	}
	teamIDs := map[string]bool{}
	teamIPs := map[string]bool{}
	for _, t := range teams {
		if t.ID == "" {
			errs = append(errs, "team.id is required")
		} else if teamIDs[t.ID] {
			errs = append(errs, "duplicate team.id: "+t.ID)
		} else {
			teamIDs[t.ID] = true
		}
		if t.IPAddress == "" {
			errs = append(errs, fmt.Sprintf("team %s: ip_address is required", t.ID))
		} else if !ipRe.MatchString(t.IPAddress) {
			errs = append(errs, fmt.Sprintf("team %s: ip_address must be IPv4", t.ID))
		} else if teamIPs[t.IPAddress] {
			errs = append(errs, "duplicate ip_address: "+t.IPAddress)
		} else {
			teamIPs[t.IPAddress] = true
		}
	}

	if len(checkers) == 0 {
		errs = append(errs, "at least one checker is required")
	}
	chkIDs := map[string]bool{}
	for _, c := range checkers {
		cid := normalizeID(c.ID)
		if cid == "" {
			errs = append(errs, "checker.id is required")
		} else if chkIDs[cid] {
			errs = append(errs, "duplicate checker.id: "+cid)
		} else {
			chkIDs[cid] = true
		}
		if c.ScriptWait < 5 {
			errs = append(errs, fmt.Sprintf("checker %s: script_wait >= 5", cid))
		}
		if c.RoundSleep < c.ScriptWait*3 {
			errs = append(errs, fmt.Sprintf("checker %s: round_sleep >= script_wait * 3", cid))
		}
		if c.ScriptRel == "" {
			errs = append(errs, fmt.Sprintf("checker %s: script_rel is required", cid))
		}
	}

	if len(errs) > 0 {
		return NewExportError(errs...)
	}
	return nil
}

func buildYAMLConfig(game GameParams, scoreboard ScoreboardParams, teams []TeamParams, checkers []CheckerParams) (string, error) {
	gameMap := yaml.Node{}
	gameMap.Kind = yaml.MappingNode
	setMapString(&gameMap, "id", game.ID)
	setMapString(&gameMap, "name", game.Name)
	setMapString(&gameMap, "start", game.StartUTC.UTC().Format("2006-01-02 15:04:05"))
	setMapString(&gameMap, "end", game.EndUTC.UTC().Format("2006-01-02 15:04:05"))
	setMapInt(&gameMap, "flag_timelive_in_min", game.FlagTTLMin)
	setMapInt(&gameMap, "basic_costs_stolen_flag_in_points", game.BasicAttackCost)
	setMapFloat(&gameMap, "cost_defence_flag_in_points", game.DefenceCost)
	if game.CoffeeBreakStartUTC != nil && game.CoffeeBreakEndUTC != nil {
		setMapString(&gameMap, "coffee_break_start", game.CoffeeBreakStartUTC.UTC().Format("2006-01-02 15:04:05"))
		setMapString(&gameMap, "coffee_break_end", game.CoffeeBreakEndUTC.UTC().Format("2006-01-02 15:04:05"))
	}

	scoreMap := yaml.Node{}
	scoreMap.Kind = yaml.MappingNode
	setMapInt(&scoreMap, "port", scoreboard.Port)
	setMapString(&scoreMap, "htmlfolder", scoreboard.HtmlFolder)
	setMapBool(&scoreMap, "random", scoreboard.Random)

	var checkerNodes []yaml.Node
	for _, c := range checkers {
		n := yaml.Node{}
		n.Kind = yaml.MappingNode
		setMapString(&n, "id", normalizeID(c.ID))
		setMapString(&n, "service_name", c.Name)
		setMapBool(&n, "enabled", c.Enabled)
		setMapString(&n, "script_path", c.ScriptRel)
		setMapInt(&n, "script_wait_in_sec", c.ScriptWait)
		setMapInt(&n, "time_sleep_between_run_scripts_in_sec", c.RoundSleep)
		checkerNodes = append(checkerNodes, n)
	}

	var teamNodes []yaml.Node
	for _, t := range teams {
		n := yaml.Node{}
		n.Kind = yaml.MappingNode
		setMapString(&n, "id", t.ID)
		setMapString(&n, "name", t.Name)
		setMapBool(&n, "active", t.Active)
		setMapString(&n, "logo", t.LogoRel)
		setMapString(&n, "ip_address", t.IPAddress)
		for k, v := range t.Ctf01dExtra {
			key := strings.TrimPrefix(k, "ctf01d_")
			setMapString(&n, key, v)
		}
		teamNodes = append(teamNodes, n)
	}

	data := yaml.Node{}
	data.Kind = yaml.MappingNode
	data.Content = append(data.Content,
		makeStringNode("game"), &gameMap,
		makeStringNode("scoreboard"), &scoreMap,
	)

	checkersSeq := yaml.Node{}
	checkersSeq.Kind = yaml.SequenceNode
	for _, cn := range checkerNodes {
		checkersSeq.Content = append(checkersSeq.Content, &cn)
	}
	data.Content = append(data.Content, makeStringNode("checkers"), &checkersSeq)

	teamsSeq := yaml.Node{}
	teamsSeq.Kind = yaml.SequenceNode
	for _, tn := range teamNodes {
		teamsSeq.Content = append(teamsSeq.Content, &tn)
	}
	data.Content = append(data.Content, makeStringNode("teams"), &teamsSeq)

	var buf bytes.Buffer
	buf.WriteString("## Combined config for ctf01d\n")
	buf.WriteString("# Auto-generated: do not edit manually; rebuild the archive instead.\n\n")

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&data); err != nil {
		return "", fmt.Errorf("encode yaml: %w", err)
	}
	enc.Close()

	return buf.String(), nil
}

func makeStringNode(s string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Value: s}
	return n
}

func setMapString(node *yaml.Node, key, value string) {
	node.Content = append(node.Content, makeStringNode(key), makeStringNode(value))
}

func setMapInt(node *yaml.Node, key string, value int) {
	node.Content = append(node.Content, makeStringNode(key), makeStringNode(strconv.Itoa(value)))
}

func setMapFloat(node *yaml.Node, key string, value float64) {
	node.Content = append(node.Content, makeStringNode(key), makeStringNode(strconv.FormatFloat(value, 'f', -1, 64)))
}

func setMapBool(node *yaml.Node, key string, value bool) {
	node.Content = append(node.Content, makeStringNode(key), makeStringNode(strconv.FormatBool(value)))
}

func ensureTeamLogos(teams []TeamParams, dataDir string, downloadsDir string) {
	for i := range teams {
		t := &teams[i]
		if strings.TrimSpace(t.LogoRel) == "" {
			t.LogoRel = fmt.Sprintf("./html/images/teams/%s.svg", safeTeamID(t.ID))
		}

		var src string
		if t.LogoSrc != "" && fileExists(t.LogoSrc) {
			src = t.LogoSrc
		} else if t.LogoURL != "" {
			if strings.HasPrefix(t.LogoURL, "http://") || strings.HasPrefix(t.LogoURL, "https://") {
				downloaded, err := downloadURLToFile(t.LogoURL, downloadsDir, safeTeamID(t.ID))
				if err == nil {
					src = downloaded
				}
			}
			if src == "" && strings.HasPrefix(t.LogoURL, "data:image") {
				written, err := writeDataURLToFile(t.LogoURL, downloadsDir, safeTeamID(t.ID))
				if err == nil {
					src = written
				}
			}
		}

		if src == "" {
			src = generateSVGLogoToFile(firstNonEmpty(t.Name, t.ID), downloadsDir, safeTeamID(t.ID))
			if strings.ToLower(path.Ext(t.LogoRel)) == ".png" {
				t.LogoRel = regexp.MustCompile(`\.png$`).ReplaceAllString(t.LogoRel, ".svg")
			}
		}

		srcExt := strings.ToLower(path.Ext(src))
		logoRelExt := strings.ToLower(path.Ext(t.LogoRel))
		if srcExt != "" && logoRelExt != srcExt {
			t.LogoRel = regexp.MustCompile(`\.[a-z0-9]+$`).ReplaceAllString(t.LogoRel, srcExt)
		}

		target := path.Join(dataDir, t.LogoRel)
		os.MkdirAll(path.Dir(target), 0o755)
		copyFile(src, target)
	}
}

func generateSVGLogoToFile(text string, dir string, preferName string) string {
	label := strings.TrimSpace(text)
	initial := "?"
	if len(label) > 0 {
		initial = strings.ToUpper(string(label[0]))
	}
	color := paletteColor(label)
	fontSize := 64
	size := 128
	svg := fmt.Sprintf(
		"<svg xmlns='http://www.w3.org/2000/svg' width='%d' height='%d'>"+
			"<rect width='100%%' height='100%%' fill='%s' />"+
			"<text x='50%%' y='56%%' dominant-baseline='middle' text-anchor='middle'"+
			" font-family='Arial, Helvetica, sans-serif' font-size='%d' fill='#fff'>%s</text>"+
			"</svg>", size, size, color, fontSize, initial)

	filePath := path.Join(dir, preferName+".svg")
	if err := os.WriteFile(filePath, []byte(svg), 0o644); err != nil {
		return ""
	}
	return filePath
}

func paletteColor(s string) string {
	palette := []string{
		"#3B82F6", "#10B981", "#F59E0B", "#EF4444", "#8B5CF6",
		"#06B6D4", "#EC4899", "#84CC16", "#F97316", "#22C55E",
	}
	h := sha256.Sum256([]byte(s))
	idx := 0
	for _, b := range h {
		idx = (idx + int(b)) % len(palette)
	}
	return palette[idx]
}

func writeDataURLToFile(dataURL string, dir string, preferName string) (string, error) {
	if m := regexp.MustCompile(`^data:(image/[a-zA-Z0-9.+\-]+);base64,(.+)$`).FindStringSubmatch(dataURL); len(m) == 3 {
		mime := m[1]
		payload := m[2]
		bytes, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return "", fmt.Errorf("invalid base64 in data URL: %w", err)
		}
		ext := extFromMIME(mime)
		filePath := path.Join(dir, preferName+ext)
		if err := os.WriteFile(filePath, bytes, 0o644); err != nil {
			return "", err
		}
		return filePath, nil
	}

	if m := regexp.MustCompile(`^data:(image/[a-zA-Z0-9.+\-]+);utf8,(.+)$`).FindStringSubmatch(dataURL); len(m) == 3 {
		mime := m[1]
		encoded := m[2]
		raw, err := url.QueryUnescape(encoded)
		if err != nil {
			return "", fmt.Errorf("invalid utf8 data URL: %w", err)
		}
		ext := extFromMIME(mime)
		filePath := path.Join(dir, preferName+ext)
		if err := os.WriteFile(filePath, []byte(raw), 0o644); err != nil {
			return "", err
		}
		return filePath, nil
	}

	return "", fmt.Errorf("invalid data:image URL")
}

func checkDownloadURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}
	host := u.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolving host: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("blocked address: %s", ip)
		}
		if ip4 := ip.To4(); ip4 != nil {
			if ip4[0] == 10 || (ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) || (ip4[0] == 192 && ip4[1] == 168) {
				return fmt.Errorf("blocked private address: %s", ip)
			}
			if ip4[0] == 169 && ip4[1] == 254 {
				return fmt.Errorf("blocked link-local address: %s", ip)
			}
			if ip4[0] == 0 {
				return fmt.Errorf("blocked address: %s", ip)
			}
		}
	}
	return nil
}

func downloadURLToFile(rawURL string, dir string, preferName string) (string, error) {
	if err := checkDownloadURL(rawURL); err != nil {
		return "", fmt.Errorf("blocked URL: %w", err)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("download logo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download logo: HTTP %d", resp.StatusCode)
	}

	maxBytes := int64(5 * 1024 * 1024)
	lr := &io.LimitedReader{R: resp.Body, N: maxBytes}
	tmpPath := path.Join(dir, preferName+".part")
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, lr); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	mime := resp.Header.Get("Content-Type")
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = mime[:idx]
	}
	ext := extFromMIME(mime)
	finalPath := path.Join(dir, preferName+ext)
	os.Rename(tmpPath, finalPath)
	return finalPath, nil
}

func extFromMIME(mime string) string {
	switch strings.TrimSpace(strings.ToLower(mime)) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/svg+xml":
		return ".svg"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

func materializeCheckers(checkers []CheckerParams, dataDir string) {
	for _, c := range checkers {
		cid := normalizeID(c.ID)
		dir := path.Join(dataDir, "checker_"+cid)
		os.MkdirAll(dir, 0o755)

		if c.BundlePath != "" && c.CheckerFromBundle {
			extracted := extractCheckerDirFromBundle(c.BundlePath, dir)
			if !extracted {
				writeDummyChecker(dir, cid)
			}
			continue
		}
		if c.BundlePath != "" && !c.CheckerFromBundle {
			writeDummyChecker(dir, cid)
			continue
		}

		files := c.Files
		if len(files) == 0 {
			files = []CheckerFile{{Src: "", Rel: "checker.py"}}
		}
		for _, f := range files {
			rel := f.Rel
			if rel == "" && f.Src != "" && fileExists(f.Src) {
				rel = path.Base(f.Src)
			}
			if rel == "" {
				rel = "checker.py"
			}
			dest := safeJoin(dir, rel)
			os.MkdirAll(path.Dir(dest), 0o755)
			if f.Src != "" && fileExists(f.Src) {
				copyFile(f.Src, dest)
			} else {
				content := fmt.Sprintf("#!/usr/bin/env python3\nprint('dummy checker for %s')\n", cid)
				os.WriteFile(dest, []byte(content), 0o644)
			}
		}
	}
}

func extractCheckerDirFromBundle(bundlePath string, destDir string) bool {
	r, err := os.Open(bundlePath)
	if err != nil {
		return false
	}
	defer r.Close()

	fi, err := r.Stat()
	if err != nil {
		return false
	}

	zr, err := zip.NewReader(r, fi.Size())
	if err != nil {
		return false
	}

	extracted := false
	for _, f := range zr.File {
		name := f.Name
		if !containsCheckerDir(name) {
			continue
		}
		extracted = true
		idx := findCheckerDirIndex(name)
		rel := name[idx:]
		if rel == "" {
			continue
		}
		target := safeJoin(destDir, rel)
		if strings.HasSuffix(name, "/") {
			os.MkdirAll(target, 0o755)
			continue
		}
		os.MkdirAll(path.Dir(target), 0o755)
		rc, err := f.Open()
		if err != nil {
			continue
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			continue
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	return extracted
}

func containsCheckerDir(name string) bool {
	for {
		idx := strings.Index(name, "checker/")
		if idx < 0 {
			return false
		}
		return true
	}
}

func writeDummyChecker(destDir string, cid string) {
	p := path.Join(destDir, "checker.py")
	if fileExists(p) {
		return
	}
	content := fmt.Sprintf("#!/usr/bin/env python3\nprint('dummy checker for %s')\n", cid)
	os.WriteFile(p, []byte(content), 0o644)
}

func materializeServiceArchives(checkers []CheckerParams, rootDir string) {
	dir := path.Join(rootDir, "archives", "services")
	for _, c := range checkers {
		if c.BundlePath == "" || !fileExists(c.BundlePath) {
			continue
		}
		os.MkdirAll(dir, 0o755)
		cid := normalizeID(c.ID)
		dest := path.Join(dir, cid+".zip")
		copyFile(c.BundlePath, dest)
	}
}

func copyTree(src string, dst string) {
	if !dirExists(src) {
		return
	}
	os.MkdirAll(dst, 0o755)
	entries, err := os.ReadDir(src)
	if err != nil {
		return
	}
	for _, entry := range entries {
		s := path.Join(src, entry.Name())
		d := path.Join(dst, entry.Name())
		if entry.IsDir() {
			copyTree(s, d)
		} else {
			os.MkdirAll(path.Dir(d), 0o755)
			copyFile(s, d)
		}
	}
}

func buildFallbackHTML(tmpdir string) string {
	dir := path.Join(tmpdir, "fallback_html")
	os.MkdirAll(dir, 0o755)

	indexHTML := `<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <title>ctf01d scoreboard</title>
  </head>
  <body>
    <h1>ctf01d scoreboard placeholder</h1>
    <p>HTML not found in repository; generated default template.</p>
  </body>
</html>`
	os.WriteFile(path.Join(dir, "index-template.html"), []byte(indexHTML), 0o644)

	teamsDir := path.Join(dir, "images", "teams")
	os.MkdirAll(teamsDir, 0o755)

	minPNG, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+ZzoAAAAASUVORK5CYII=")
	for i := 1; i <= 10; i++ {
		os.WriteFile(path.Join(teamsDir, fmt.Sprintf("team%02d.png", i)), minPNG, 0o644)
	}

	return dir
}

func composeYML(scoreboard ScoreboardParams, options Options) string {
	project := options.ComposeProject
	return fmt.Sprintf(`version: '3'

services:
  ctf01d_jury:
    container_name: ctf01d_jury_%s
    image: sea5kg/ctf01d:latest
    volumes:
      - "./data:/usr/share/ctf01d"
    environment:
      CTF01D_WORKDIR: "/usr/share/ctf01d"
    ports:
      - "%d:%d"
    networks:
      - ctf01d_net

networks:
  ctf01d_net:
    driver: bridge
`, project, scoreboard.Port, scoreboard.Port)
}

func packZip(rootDir string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	base := path.Base(rootDir)
	parent := path.Dir(rootDir)

	err := filepathWalk(parent, base, func(relPath string, info os.FileInfo) error {
		if info.IsDir() {
			_, err := w.Create(relPath + "/")
			return err
		}
		fw, err := w.Create(relPath)
		if err != nil {
			return err
		}
		fullPath := path.Join(parent, relPath)
		f, err := os.Open(fullPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, f)
		f.Close()
		return err
	})
	if err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func filepathWalk(parent string, base string, fn func(string, os.FileInfo) error) error {
	root := path.Join(parent, base)
	return walkDir(root, base, fn)
}

func walkDir(dir string, prefix string, fn func(string, os.FileInfo) error) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		relPath := prefix + "/" + entry.Name()
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if err := fn(relPath, info); err != nil {
			return err
		}
		if entry.IsDir() {
			if err := walkDir(path.Join(dir, entry.Name()), relPath, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeID(val string) string {
	s := strings.ToLower(val)
	s = nonAlphaNum.ReplaceAllString(s, "_")
	s = trailingUnd.ReplaceAllString(s, "")
	return s
}

func safeTeamID(val string) string {
	return normalizeID(val)
}

func safeJoin(base string, rel string) string {
	clean := strings.ReplaceAll(rel, "\\", "/")
	clean = strings.TrimLeft(clean, "/")
	if clean == "" || strings.Contains(clean, "\x00") {
		return path.Join(base, "unknown")
	}
	segments := strings.Split(clean, "/")
	for _, s := range segments {
		if s == "" || s == "." || s == ".." {
			return path.Join(base, "unknown")
		}
	}
	return path.Join(base, path.Join(segments...))
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	os.MkdirAll(path.Dir(dst), 0o755)
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
