package ctf01d

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
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
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	gameIDRe    = regexp.MustCompile(`^[a-z0-9]+$`)
	ipRe        = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)
	trailingUnd = regexp.MustCompile(`^_+|_+$`)
)

const (
	dirMode         = 0o755
	privateFileMode = 0o600

	randomSuffixBytes    = 4
	defaultScriptWait    = 10
	roundSleepMultiplier = 3
	defaultRoundSleep    = defaultScriptWait * roundSleepMultiplier
	checkerDirName       = "checker/"
	ctf01dYAMLIndent     = 2
	dataURLMatchCount    = 3

	minFlagTTLMin      = 1
	maxFlagTTLMin      = 25
	minBasicAttackCost = 1
	maxBasicAttackCost = 500
	defaultFlagTTLMin  = 10
	defaultAttackCost  = 100
	defaultDefenceCost = 50.0
	minScoreboardPort  = 11
	maxScoreboardPort  = 65535
	defaultScorePort   = 8080
	minScriptWait      = 5
	minutesPerHour     = 60

	logoTextFontSize   = 64
	logoImageSize      = 128
	logoFetchTimeout   = 15 * time.Second
	maxHTTPRedirects   = 10
	maxLogoBytes       = 5 * 1024 * 1024
	maxExtractFileSize = 200 * 1024 * 1024
	fallbackLogoCount  = 10

	pngExt = ".png"
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
	if err := os.MkdirAll(dataDir, dirMode); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	htmlSource := options.HtmlSourcePath
	if options.IncludeHTML {
		if htmlSource == "" || !dirExists(htmlSource) {
			htmlSource, err = buildFallbackHTML(tmpDir)
			if err != nil {
				return nil, fmt.Errorf("build fallback html: %w", err)
			}
		} else {
			abs, err := filepath.Abs(htmlSource)
			if err != nil {
				return nil, fmt.Errorf("invalid html_source_path: %w", err)
			}
			clean := filepath.Clean(abs)
			if strings.Contains(clean, "..") {
				return nil, errors.New("invalid html_source_path: must not contain '..' components")
			}
			htmlSource = clean
		}
		if err := copyTree(htmlSource, path.Join(dataDir, "html")); err != nil {
			return nil, fmt.Errorf("copy html: %w", err)
		}
	}

	downloadsDir := path.Join(tmpDir, "downloads")
	if err := os.MkdirAll(downloadsDir, dirMode); err != nil {
		return nil, fmt.Errorf("create downloads dir: %w", err)
	}
	if err := ensureTeamLogos(teams, dataDir, downloadsDir); err != nil {
		return nil, fmt.Errorf("prepare team logos: %w", err)
	}

	if err := materializeCheckers(checkers, dataDir); err != nil {
		return nil, fmt.Errorf("materialize checkers: %w", err)
	}
	if err := materializeServiceArchives(checkers, root); err != nil {
		return nil, fmt.Errorf("materialize service archives: %w", err)
	}

	cfgPath := path.Join(dataDir, "config.yml")
	cfgContent, err := buildYAMLConfig(game, scoreboard, teams, checkers)
	if err != nil {
		return nil, fmt.Errorf("build config: %w", err)
	}
	if err := os.WriteFile(cfgPath, []byte(cfgContent), privateFileMode); err != nil {
		return nil, fmt.Errorf("write config: %w", err)
	}

	if len(options.Warnings) > 0 {
		warningsPath := path.Join(root, "EXPORT_WARNINGS.txt")
		if err := os.WriteFile(warningsPath, []byte(strings.Join(options.Warnings, "\n")), privateFileMode); err != nil {
			return nil, fmt.Errorf("write warnings: %w", err)
		}
	}

	if options.IncludeCompose {
		composePath := path.Join(root, "docker-compose.yml")
		if err := os.WriteFile(composePath, []byte(composeYML(scoreboard, options)), privateFileMode); err != nil {
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
		b := make([]byte, randomSuffixBytes)
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
			c.ScriptWait = defaultScriptWait
		}
		if c.RoundSleep < c.ScriptWait*roundSleepMultiplier {
			c.RoundSleep = c.ScriptWait * roundSleepMultiplier
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
	switch {
	case len(preferred) > 0:
		pick = preferred
	case len(topLevel) > 0:
		pick = topLevel
	default:
		pick = relFiles
	}
	if len(pick) > 0 {
		return "./" + pick[0]
	}
	return ""
}

func findCheckerDirIndex(name string) int {
	for {
		idx := strings.Index(name, checkerDirName)
		if idx < 0 {
			return -1
		}
		rel := name[idx+len(checkerDirName):]
		if rel != "" {
			return idx + len(checkerDirName)
		}
		remaining := name[idx+len(checkerDirName):]
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

//nolint:gocyclo // validation is intentionally a flat list of request-field checks.
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
	if game.FlagTTLMin < minFlagTTLMin || game.FlagTTLMin > maxFlagTTLMin {
		errs = append(errs, "game.flag_ttl_min must be between 1 and 25")
	}
	if game.BasicAttackCost < minBasicAttackCost || game.BasicAttackCost > maxBasicAttackCost {
		errs = append(errs, "game.basic_attack_cost must be between 1 and 500")
	}

	if scoreboard.Port < minScoreboardPort || scoreboard.Port > maxScoreboardPort {
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
		switch {
		case t.ID == "":
			errs = append(errs, "team.id is required")
		case teamIDs[t.ID]:
			errs = append(errs, "duplicate team.id: "+t.ID)
		default:
			teamIDs[t.ID] = true
		}
		switch {
		case t.IPAddress == "":
			errs = append(errs, fmt.Sprintf("team %s: ip_address is required", t.ID))
		case !ipRe.MatchString(t.IPAddress):
			errs = append(errs, fmt.Sprintf("team %s: ip_address must be IPv4", t.ID))
		case teamIPs[t.IPAddress]:
			errs = append(errs, "duplicate ip_address: "+t.IPAddress)
		default:
			teamIPs[t.IPAddress] = true
		}
	}

	if len(checkers) == 0 {
		errs = append(errs, "at least one checker is required")
	}
	chkIDs := map[string]bool{}
	for _, c := range checkers {
		cid := normalizeID(c.ID)
		switch {
		case cid == "":
			errs = append(errs, "checker.id is required")
		case chkIDs[cid]:
			errs = append(errs, "duplicate checker.id: "+cid)
		default:
			chkIDs[cid] = true
		}
		if c.ScriptWait < minScriptWait {
			errs = append(errs, fmt.Sprintf("checker %s: script_wait >= 5", cid))
		}
		if c.RoundSleep < c.ScriptWait*roundSleepMultiplier {
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
	setMapFloat(&gameMap, "cost_defense_flag_in_points", game.DefenceCost)
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
	enc.SetIndent(ctf01dYAMLIndent)
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

func ensureTeamLogos(teams []TeamParams, dataDir string, downloadsDir string) error {
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
			written, err := generateSVGLogoToFile(firstNonEmpty(t.Name, t.ID), downloadsDir, safeTeamID(t.ID))
			if err != nil {
				return err
			}
			src = written
			if strings.ToLower(path.Ext(t.LogoRel)) == pngExt {
				t.LogoRel = regexp.MustCompile(`\.png$`).ReplaceAllString(t.LogoRel, ".svg")
			}
		}

		srcExt := strings.ToLower(path.Ext(src))
		logoRelExt := strings.ToLower(path.Ext(t.LogoRel))
		if srcExt != "" && logoRelExt != srcExt {
			t.LogoRel = regexp.MustCompile(`\.[a-z0-9]+$`).ReplaceAllString(t.LogoRel, srcExt)
		}

		target := path.Join(dataDir, t.LogoRel)
		if err := os.MkdirAll(path.Dir(target), dirMode); err != nil {
			return err
		}
		if err := copyFile(src, target); err != nil {
			return err
		}
	}
	return nil
}

func generateSVGLogoToFile(text string, dir string, preferName string) (string, error) {
	label := strings.TrimSpace(text)
	initial := "?"
	if len(label) > 0 {
		initial = strings.ToUpper(string(label[0]))
	}
	color := paletteColor(label)
	svg := fmt.Sprintf(
		"<svg xmlns='http://www.w3.org/2000/svg' width='%d' height='%d'>"+
			"<rect width='100%%' height='100%%' fill='%s' />"+
			"<text x='50%%' y='56%%' dominant-baseline='middle' text-anchor='middle'"+
			" font-family='Arial, Helvetica, sans-serif' font-size='%d' fill='#fff'>%s</text>"+
			"</svg>", logoImageSize, logoImageSize, color, logoTextFontSize, xmlEscape(initial))

	filePath := path.Join(dir, preferName+".svg")
	if err := os.WriteFile(filePath, []byte(svg), privateFileMode); err != nil {
		return "", err
	}
	return filePath, nil
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	xml.Escape(&buf, []byte(s))
	return buf.String()
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
	if m := regexp.MustCompile(`^data:(image/[a-zA-Z0-9.+\-]+);base64,(.+)$`).FindStringSubmatch(dataURL); len(m) == dataURLMatchCount {
		mime := m[1]
		payload := m[2]
		decoded, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return "", fmt.Errorf("invalid base64 in data URL: %w", err)
		}
		ext := extFromMIME(mime)
		filePath := path.Join(dir, preferName+ext)
		if err := os.WriteFile(filePath, decoded, privateFileMode); err != nil {
			return "", err
		}
		return filePath, nil
	}

	if m := regexp.MustCompile(`^data:(image/[a-zA-Z0-9.+\-]+);utf8,(.+)$`).FindStringSubmatch(dataURL); len(m) == dataURLMatchCount {
		mime := m[1]
		encoded := m[2]
		raw, err := url.QueryUnescape(encoded)
		if err != nil {
			return "", fmt.Errorf("invalid utf8 data URL: %w", err)
		}
		ext := extFromMIME(mime)
		filePath := path.Join(dir, preferName+ext)
		if err := os.WriteFile(filePath, []byte(raw), privateFileMode); err != nil {
			return "", err
		}
		return filePath, nil
	}

	return "", errors.New("invalid data:image URL")
}

var exporterBlockedNets []*net.IPNet
var exporterBlockedNetsOnce sync.Once

func initExporterBlockedNets() {
	exporterBlockedNetsOnce.Do(func() {
		exporterBlockedNets = mustParseExporterCIDRs(
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

func mustParseExporterCIDRs(cidrs ...string) []*net.IPNet {
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
	initExporterBlockedNets()
	for _, n := range exporterBlockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func checkDownloadURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("URL has no host")
	}
	return nil
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
		if isBlockedIP(ip.IP) {
			return nil, fmt.Errorf("blocked address: %s", ip.IP)
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

func downloadURLToFile(rawURL string, dir string, preferName string) (string, error) {
	if err := checkDownloadURL(rawURL); err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	client := &http.Client{
		Timeout: logoFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxHTTPRedirects {
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
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("create logo request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download logo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download logo: HTTP %d", resp.StatusCode)
	}

	lr := &io.LimitedReader{R: resp.Body, N: maxLogoBytes}
	tmpPath := path.Join(dir, preferName+".part")
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, lr); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	mime := resp.Header.Get("Content-Type")
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = mime[:idx]
	}
	ext := extFromMIME(mime)
	finalPath := path.Join(dir, preferName+ext)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return "", err
	}
	return finalPath, nil
}

func extFromMIME(mime string) string {
	switch strings.TrimSpace(strings.ToLower(mime)) {
	case "image/png":
		return pngExt
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/svg+xml":
		return ".svg"
	case "image/gif":
		return ".gif"
	default:
		return pngExt
	}
}

func materializeCheckers(checkers []CheckerParams, dataDir string) error {
	for _, c := range checkers {
		cid := normalizeID(c.ID)
		dir := path.Join(dataDir, "checker_"+cid)
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return err
		}

		if c.BundlePath != "" && c.CheckerFromBundle {
			extracted, err := extractCheckerDirFromBundle(c.BundlePath, dir)
			if err != nil {
				return err
			}
			if !extracted {
				if err := writeDummyChecker(dir, cid); err != nil {
					return err
				}
			}
			continue
		}
		if c.BundlePath != "" && !c.CheckerFromBundle {
			if err := writeDummyChecker(dir, cid); err != nil {
				return err
			}
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
			if err := os.MkdirAll(path.Dir(dest), dirMode); err != nil {
				return err
			}
			if f.Src != "" && fileExists(f.Src) {
				if err := copyFile(f.Src, dest); err != nil {
					return err
				}
			} else {
				content := fmt.Sprintf("#!/usr/bin/env python3\nprint('dummy checker for %s')\n", cid)
				if err := os.WriteFile(dest, []byte(content), privateFileMode); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func extractCheckerDirFromBundle(bundlePath string, destDir string) (bool, error) {
	r, err := os.Open(bundlePath)
	if err != nil {
		return false, nil
	}
	defer r.Close()

	fi, err := r.Stat()
	if err != nil {
		return false, nil
	}

	zr, err := zip.NewReader(r, fi.Size())
	if err != nil {
		return false, nil
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
			if err := os.MkdirAll(target, dirMode); err != nil {
				return false, err
			}
			continue
		}
		if err := os.MkdirAll(path.Dir(target), dirMode); err != nil {
			return false, err
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		out, err := os.Create(target)
		if err != nil {
			_ = rc.Close()
			continue
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, maxExtractFileSize))
		closeOutErr := out.Close()
		closeRCErr := rc.Close()
		if copyErr != nil {
			return false, copyErr
		}
		if closeOutErr != nil {
			return false, closeOutErr
		}
		if closeRCErr != nil {
			return false, closeRCErr
		}
	}
	return extracted, nil
}

func containsCheckerDir(name string) bool {
	return strings.Contains(name, checkerDirName)
}

func writeDummyChecker(destDir string, cid string) error {
	p := path.Join(destDir, "checker.py")
	if fileExists(p) {
		return nil
	}
	content := fmt.Sprintf("#!/usr/bin/env python3\nprint('dummy checker for %s')\n", cid)
	return os.WriteFile(p, []byte(content), privateFileMode)
}

func materializeServiceArchives(checkers []CheckerParams, rootDir string) error {
	dir := path.Join(rootDir, "archives", "services")
	for _, c := range checkers {
		if c.BundlePath == "" || !fileExists(c.BundlePath) {
			continue
		}
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return err
		}
		cid := normalizeID(c.ID)
		dest := path.Join(dir, cid+".zip")
		if err := copyFile(c.BundlePath, dest); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(src string, dst string) error {
	if !dirExists(src) {
		return nil
	}
	if err := os.MkdirAll(dst, dirMode); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		s := path.Join(src, entry.Name())
		d := path.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyTree(s, d); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(path.Dir(d), dirMode); err != nil {
				return err
			}
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildFallbackHTML(tmpdir string) (string, error) {
	dir := path.Join(tmpdir, "fallback_html")
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", err
	}

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
	if err := os.WriteFile(path.Join(dir, "index-template.html"), []byte(indexHTML), privateFileMode); err != nil {
		return "", err
	}

	teamsDir := path.Join(dir, "images", "teams")
	if err := os.MkdirAll(teamsDir, dirMode); err != nil {
		return "", err
	}

	minPNG, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+ZzoAAAAASUVORK5CYII=")
	if err != nil {
		return "", err
	}
	for i := 1; i <= fallbackLogoCount; i++ {
		if err := os.WriteFile(path.Join(teamsDir, fmt.Sprintf("team%02d.png", i)), minPNG, privateFileMode); err != nil {
			return "", err
		}
	}

	return dir, nil
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
		_, copyErr := io.Copy(fw, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
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
	if err := os.MkdirAll(path.Dir(dst), dirMode); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
