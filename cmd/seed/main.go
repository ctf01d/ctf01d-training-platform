package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/ctf01d/ctf01d-training-platform/internal/config"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/pkg/logger"
)

func main() {
	if err := seed(); err != nil {
		fmt.Fprintf(os.Stderr, "seed error: %v\n", err)
		os.Exit(1)
	}
}

func seed() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log, err := logger.New(cfg.Env, cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}
	defer logger.Sync(log)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	store, err := repository.NewStore(ctx, cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer store.Close()

	q := store.Queries

	adminPassword := os.Getenv("SEED_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin12345"
	}

	adminUser, created, err := seedAdmin(ctx, q, adminPassword)
	if err != nil {
		return fmt.Errorf("seeding admin: %w", err)
	}
	logSeed(log, "admin user", adminUser.UserName, created)

	player1, created, err := seedUser(ctx, q, "player1", "Player One", "player", "player123")
	if err != nil {
		return fmt.Errorf("seeding player1: %w", err)
	}
	logSeed(log, "user", player1.UserName, created)

	player2, created, err := seedUser(ctx, q, "player2", "Player Two", "player", "player123")
	if err != nil {
		return fmt.Errorf("seeding player2: %w", err)
	}
	logSeed(log, "user", player2.UserName, created)

	// Real SibirCTF / CyberSibir historical dataset, ported from the legacy
	// Rails db/seeds.rb: editions by year, real CTFtime scoreboards, rosters,
	// Siberian universities, services and final standings.
	if err := seedSibir(ctx, q, log); err != nil {
		return fmt.Errorf("seeding SibirCTF data: %w", err)
	}

	log.Info("seed completed successfully")
	return nil
}

// -----------------------------------------------------------------------------
// SibirCTF / CyberSibir dataset (legacy Rails db/seeds.rb port)

var palette = []string{
	"#3B82F6", "#10B981", "#F59E0B", "#EF4444", "#8B5CF6",
	"#06B6D4", "#EC4899", "#84CC16", "#F97316", "#22C55E",
}

type ctftimeTeamInfo struct {
	ID          string
	Country     string
	AvatarURL   string
	Academic    string
	AcademicURL string
}

func (i ctftimeTeamInfo) profileURL() string {
	if i.ID == "" {
		return ""
	}
	return "https://ctftime.org/team/" + i.ID + "/"
}

func seedSibir(ctx context.Context, q *db.Queries, log *zap.Logger) error {
	rng := rand.New(rand.NewSource(42)) //nolint:gosec // deterministic seed data, not security-sensitive
	now := time.Now()
	color := func() string { return palette[rng.Intn(len(palette))] }

	// --- Universities ---------------------------------------------------------
	const (
		uniMSU     = "Московский государственный университет имени М.В. Ломоносова"
		uniSPbU    = "Санкт-Петербургский государственный университет"
		uniNSU     = "Новосибирский государственный университет (НГУ, NSU)"
		uniMIPT    = "Московский физико-технический институт"
		uniMEPhI   = "Национальный исследовательский ядерный университет МИФИ"
		uniAltSTU  = "Алтайский государственный технический университет (АлтГТУ, AltSTU)"
		uniTUSUR   = "Томский государственный университет систем управления и радиоэлектроники (ТУСУР, TUSUR)"
		uniTSU     = "Томский государственный университет (ТГУ, TSU)"
		uniUrFU    = "Уральский федеральный университет"
		uniKFU     = "Казанский федеральный университет"
		uniFEFU    = "Дальневосточный федеральный университет"
		uniSSUGT   = "Сибирский государственный университет геосистем и технологий (ССУГиТ, SSUGT)"
		uniNSTU    = "Новосибирский государственный технический университет (НГТУ, NSTU)"
		uniITMO    = "Университет ИТМО"
		uniCentral = "Центральный университет"
		uniFSO     = "Академия Федеральной службы охраны Российской Федерации"
		uniMIREA   = "МИРЭА — Российский технологический университет (MIREA)"
		uniOmSTU   = "Омский государственный технический университет (ОмГТУ, OmSTU)"
		uniOmskAT  = "Омский авиационный колледж (Omsk Aviation College)"
		uniSibSU   = "Сибирский государственный университет науки и технологий имени академика М.Ф. Решетнева (СибГУ, SibSU)"

		uniKuzSTU = "Кузбасский государственный технический университет имени Т.Ф. Горбачёва (КузГТУ, KuzSTU)"
		uniAltSU  = "Алтайский государственный университет (АлтГУ, AltSU)"
	)
	universityNames := []string{
		uniMSU,
		uniSPbU,
		uniNSU,
		uniMIPT,
		uniMEPhI,
		"Бауманский МГТУ",
		uniITMO,
		uniUrFU,
		uniKFU,
		uniFEFU,
		"Высшая школа экономики",
		uniAltSTU,
		uniTUSUR,
		uniTSU,
		uniSSUGT,
		uniNSTU,
		uniCentral,
		uniFSO,
		uniMIREA,
		uniOmSTU,
		uniOmskAT,
		uniSibSU,
		uniKuzSTU,
		uniAltSU,
	}
	// Real logos for select universities (served from web/public/img/...).
	universityLogos := map[string]string{
		uniMSU:     "/img/university-logos/msu.jpg",
		uniSPbU:    "/img/university-logos/spbgu.png",
		uniNSU:     "/img/university-logos/nsu.png",
		uniMIPT:    "/img/university-logos/mipt.png",
		uniMEPhI:   "/img/university-logos/mephi.png",
		uniITMO:    "/img/university-logos/itmo.png",
		uniUrFU:    "/img/university-logos/urfu.jpg",
		uniKFU:     "/img/university-logos/kfu.jpg",
		uniFEFU:    "/img/university-logos/fefu.png",
		uniAltSTU:  "/img/university-logos/altstu.png",
		uniTUSUR:   "/img/university-logos/tusur.jpg",
		uniTSU:     "/img/university-logos/tsu.png",
		uniNSTU:    "/img/university-logos/nstu.jpg",
		uniCentral: "/img/university-logos/central-university.jpg",
		uniMIREA:   "/img/university-logos/mirea.png",
		uniOmSTU:   "/img/university-logos/omstu.png",
		uniOmskAT:  "/img/university-logos/oak.jpg",
		uniSibSU:   "/img/university-logos/sibsu.png",
		uniSSUGT:   "/img/university-logos/ssugt.jpg",
	}
	universitySites := map[string]string{
		uniAltSTU:  "http://www.altstu.ru/",
		uniTUSUR:   "https://tusur.ru/",
		uniTSU:     "http://tsu.ru/",
		uniSSUGT:   "https://sgugit.ru/en/",
		uniNSTU:    "http://nstu.ru/",
		uniCentral: "https://centraluniversity.ru/",
		uniMIREA:   "https://www.mirea.ru/",
		uniOmSTU:   "https://omgtu.ru/",
		uniOmskAT:  "http://www.oatctf.ru/",
		uniSibSU:   "http://sibsau.ru/",
		uniKuzSTU:  "https://kuzstu.ru/",
		uniAltSU:   "https://www.asu.ru/",
	}
	uniByName := map[string]db.University{}
	for _, name := range universityNames {
		logo, hasLogo := universityLogos[name]
		site := universitySites[name]
		avatar := svgAvatar(name, color())
		if hasLogo {
			avatar = logo
		}
		uni, created, err := getOrCreateUniversity(ctx, q, name, site, avatar)
		if err != nil {
			return fmt.Errorf("university %q: %w", name, err)
		}
		if !created {
			params := db.UpdateUniversityParams{ID: uni.ID}
			if site != "" && (uni.SiteUrl == nil || *uni.SiteUrl == "") {
				params.SiteUrl = &site
			}
			// Backfill the logo on universities seeded before logos existed.
			if hasLogo && (uni.AvatarUrl == nil || *uni.AvatarUrl != logo) {
				params.AvatarUrl = &logo
			}
			if params.SiteUrl != nil || params.AvatarUrl != nil {
				uni, err = q.UpdateUniversity(ctx, params)
				if err != nil {
					return fmt.Errorf("university %q backfill: %w", name, err)
				}
			}
		}
		logSeed(log, "university", name, created)
		uniByName[name] = uni
	}

	// --- Teams ----------------------------------------------------------------
	teamByName := map[string]db.Team{}
	ctftimeTeams := map[string]ctftimeTeamInfo{
		"!2day": {ID: "36203"},
		"(_xXx_-=HOBOCu6uPCKuE_IICbl_1337=-_xXx_)": {ID: "21140"},
		"4Ray":                       {ID: "281722", Country: "RU", AvatarURL: "https://ctftime.org/media/team/2._4ray_Logo_WHITE_1.jpg", Academic: "Russian Technological University MIREA", AcademicURL: "https://www.mirea.ru/"},
		"4ерниkа":                    {ID: "275918"},
		"a-cool-team":                {ID: "275919"},
		"A4PT Reshetneva":            {ID: "273522"},
		"BANOЧKA":                    {ID: "275920"},
		"CatchFM":                    {ID: "32132", Country: "RU", AvatarURL: "https://ctftime.org/media/team/catchFM_logo.png"},
		"CubaLibre":                  {ID: "275916"},
		"CUT":                        {ID: "358931", Country: "RU", AvatarURL: "https://ctftime.org/media/team/scissors.png", Academic: "Central University", AcademicURL: "https://centraluniversity.ru/"},
		"CyberCringe":                {ID: "277834"},
		"CyberPatriots":              {ID: "8471"},
		"d34dl1n3":                   {ID: "35562", Country: "RU", AvatarURL: "https://ctftime.org/media/team/128_1.png", Academic: "SSUGT", AcademicURL: "https://sgugit.ru/en/"},
		"datapoison":                 {ID: "179977", Country: "RU", AvatarURL: "https://ctftime.org/media/team/IMG_20220723_201337_063.png"},
		"Dragon Hat":                 {ID: "49385", Country: "RU", AvatarURL: "https://ctftime.org/media/team/dragon_hat_2.jpg"},
		"FoXXXeS":                    {ID: "45678"},
		"Ibeee":                      {ID: "186804", Country: "RU", AvatarURL: "https://ctftime.org/media/team/35168CC6-A92A-43AE-8E34-E42658E25B46.jpeg"},
		"kekw":                       {ID: "118874"},
		"Keva19":                     {ID: "105731", Country: "RU"},
		"LCD":                        {ID: "270230", Country: "RU", AvatarURL: "https://ctftime.org/media/team/photo_2025-01-13_06-47-07.jpg"},
		"Life":                       {ID: "8625"},
		"Mu574n9":                    {ID: "45677"},
		"N0N@me13":                   {ID: "209571", Country: "RU", AvatarURL: "https://ctftime.org/media/team/nn13_main.jpg"},
		"NetOverkill":                {ID: "360551"},
		"NFB":                        {ID: "202829", Country: "RU", AvatarURL: "https://ctftime.org/media/team/duck_50.png", Academic: "OmSTU", AcademicURL: "https://omgtu.ru/"},
		"o1d_bu7_go1d":               {ID: "213673", Country: "RU", AvatarURL: "https://ctftime.org/media/team/obg-logo.png"},
		"Omaviat":                    {ID: "49106", Country: "RU", AvatarURL: "https://ctftime.org/media/team/logo_303.png", Academic: "Omsk Aviation College", AcademicURL: "http://www.oatctf.ru/"},
		"paperwhale":                 {ID: "27229", Country: "RU", AvatarURL: "https://ctftime.org/media/team/Dhq42M16goE.jpg", Academic: "SibSU", AcademicURL: "http://sibsau.ru/"},
		"QarabagTeam":                {ID: "186802", Country: "RU", AvatarURL: "https://ctftime.org/media/team/%D0%9A%D0%BE%D0%BD%D1%8C.png", Academic: "NSTU", AcademicURL: "http://nstu.ru/"},
		"R3T4RD0Z":                   {ID: "380155"},
		"rwx":                        {ID: "4351", Country: "RU", AvatarURL: "https://ctftime.org/media/team/rwx.jpg"},
		"ScareCrow":                  {ID: "12515"},
		"SharLike":                   {ID: "16172", Country: "RU", AvatarURL: "https://ctftime.org/media/team/sharlike.jpg", Academic: "AltSTU", AcademicURL: "http://www.altstu.ru/"},
		"SharNear":                   {ID: "105950"},
		"SiBears":                    {ID: "557", Country: "RU", AvatarURL: "https://ctftime.org/media/team/sibears.jpg", Academic: "TSU", AcademicURL: "http://tsu.ru/"},
		"smiley-from-telega":         {ID: "170324", Country: "RU", AvatarURL: "https://ctftime.org/media/team/2024-12-22_10.50.57.jpg"},
		"Tanuki squad":               {ID: "76462", Country: "RU", AvatarURL: "https://ctftime.org/media/team/Bezymyanny.png"},
		"Team 16":                    {ID: "151961"},
		"Team Information Offensive": {ID: "27397"},
		"TyumGUard":                  {ID: "380152"},
		"UkVQ":                       {ID: "104638", Country: "RU", Academic: "Tomsk State University of Control Systems and Radioelectronics (TUSUR)", AcademicURL: "https://tusur.ru/"},
		"vim>nano":                   {ID: "368995"},
		"W@zz4b1":                    {ID: "358319"},
		"XAKCET":                     {ID: "378647"},
		"xXx_Я_не_ХЛЕБ_я_КОТ_хХх":    {ID: "39488"},
		"Yozik":                      {ID: "1445", Country: "RU", AvatarURL: "https://ctftime.org/media/team/jo.jpg"},
		"ИнфоБесы":                   {ID: "275008"},
		"Продам гараж за флаги": {ID: "212793", Country: "RU", AvatarURL: "https://ctftime.org/media/team/logo_189.jpg"},
		"Суслобатя":             {ID: "275917"},
		"химозный рулет":        {ID: "380153"},
		"Циферки":               {ID: "380154", Country: "IT", AvatarURL: "https://ctftime.org/media/team/photo_2025-04-10_19.57.39.jpeg"},
		"ыыыыЫЫЫЫЫ":             {ID: "275915"},
	}
	// Some teams appear under different names across editions (e.g. a Russian and
	// an English spelling). canonTeam folds those aliases onto a single canonical
	// name so they resolve to one team record.
	teamAliases := map[string]string{
		"КиберПатриоты": "CyberPatriots",
		"Netoverkill":   "NetOverkill",
	}
	canonTeam := func(name string) string {
		if c, ok := teamAliases[name]; ok {
			return c
		}
		return name
	}

	// ctf01d metadata (config id / ip) per game, keyed by lower(team name).
	rosterMeta := map[string]map[string]struct{ id, ip string }{}
	addMeta := func(game, name, id, ip string) {
		name = canonTeam(name)
		m := rosterMeta[game]
		if m == nil {
			m = map[string]struct{ id, ip string }{}
			rosterMeta[game] = m
		}
		m[strings.ToLower(name)] = struct{ id, ip string }{id, ip}
	}

	ensureTeam := func(name, descr string) (db.Team, error) {
		name = canonTeam(name)
		if t, ok := teamByName[name]; ok {
			return t, nil
		}
		info := ctftimeTeams[name]
		descr = enrichTeamDescription(descr, info)
		avatar := svgAvatar(name, color())
		if info.AvatarURL != "" {
			avatar = info.AvatarURL
		}
		t, created, err := getOrCreateTeam(ctx, q, name, descr, avatar)
		if err != nil {
			return db.Team{}, err
		}
		t, err = backfillCTFtimeTeamInfo(ctx, q, t, descr, info)
		if err != nil {
			return db.Team{}, err
		}
		logSeed(log, "team", t.Name, created)
		teamByName[name] = t
		return t, nil
	}

	type rosterTeam struct{ cfg, name, ip string }

	cyber2025Roster := []rosterTeam{
		{"t01", "QarabagTeam", "10.10.1.3"}, {"t02", "W@zz4b1", "10.10.2.3"},
		{"t03", "smiley-from-telega", "10.10.3.3"}, {"t04", "R3T4RD0Z", "10.10.4.3"},
		{"t05", "химозный рулет", "10.10.5.3"}, {"t06", "SiBears", "10.10.6.3"},
		{"t07", "kekw", "10.10.7.3"}, {"t08", "CyberCringe", "10.10.8.3"},
		{"t09", "NFB", "10.10.9.3"}, {"t10", "SharLike", "10.10.10.3"},
		{"t11", "ScareCrow", "10.10.11.3"}, {"t12", "d34dl1n3", "10.10.12.3"},
		{"t14", "4Ray", "10.10.14.3"}, {"t15", "N0N@me13", "10.10.15.3"},
		{"t16", "XAKCET", "10.10.16.3"}, {"t17", "TyumGUard", "10.10.17.3"},
		{"t18", "datapoison", "10.10.18.3"}, {"t19", "Netoverkill", "10.10.19.3"},
		{"t20", "CUT", "10.10.20.3"}, {"t21", "Ibeee", "10.10.21.3"},
		{"t22", "o1d_bu7_go1d", "10.10.22.3"}, {"t23", "CyberPatriots", "10.10.23.3"},
		{"t24", "vim>nano", "10.10.24.3"}, {"t26", "Циферки", "10.10.26.3"},
	}
	for _, r := range cyber2025Roster {
		descr := fmt.Sprintf("CyberSibir 2025 roster · ctf01d_id: %s · IP: %s · Активна: yes", r.cfg, r.ip)
		if _, err := ensureTeam(r.name, descr); err != nil {
			return err
		}
		addMeta("CyberSibir 2025", r.name, r.cfg, r.ip)
	}

	cyber2026Roster := []rosterTeam{
		{"t01", "SharLike", "10.10.1.3"}, {"t02", "CUT", "10.10.2.3"},
		{"t03", "SiBears", "10.10.3.3"}, {"t04", "N0N@me13", "10.10.4.3"},
		{"t05", "4Ray", "10.10.5.3"}, {"t06", "Error Yager", "10.10.6.3"},
		{"t07", "1sk4nd3r", "10.10.7.3"}, {"t08", "Netrunners", "10.10.8.3"},
		{"t09", "ufoufo", "10.10.9.3"}, {"t10", "W@zz4bi", "10.10.10.3"},
		{"t11", "QarabagTeam", "10.10.11.3"}, {"t12", "ыыыыЫЫЫЫЫ", "10.10.12.3"},
		{"t13", "d34dl1n3", "10.10.13.3"}, {"t14", "...", "10.10.14.3"},
		{"t15", "Mustang", "10.10.15.3"}, {"t16", "КиберМамонты", "10.10.16.3"},
		{"t17", "The Power of Elijah", "10.10.17.3"}, {"t18", "Циферки", "10.10.18.3"},
		{"t19", "avek", "10.10.19.3"}, {"t20", "КиберПатриоты", "10.10.20.3"},
		{"t21", "NetOverkill", "10.10.21.3"},
	}
	for _, r := range cyber2026Roster {
		descr := fmt.Sprintf("CyberSibir 2026 roster · ctf01d_id: %s · IP: %s · Активна: yes", r.cfg, r.ip)
		if _, err := ensureTeam(r.name, descr); err != nil {
			return err
		}
		addMeta("CyberSibir 2026", r.name, r.cfg, r.ip)
	}

	sibir2018Roster := []rosterTeam{
		{"team1", "Life", "10.218.1.2"}, {"team2", "Void*", "10.218.2.2"},
		{"team3", "SiBears", "10.218.3.2"}, {"team4", "Novosibirsk SU X", "10.218.4.2"},
		{"team5", "paperwhale", "10.218.5.2"}, {"team6", "Omaviat", "10.218.6.2"},
		{"team7", "CatchFM", "10.218.7.2"}, {"team8", "RWX", "10.218.8.2"},
		{"team9", "SharLike", "10.218.9.2"}, {"team10", "d34dl1n3", "10.218.10.2"},
		{"team11", "n57u n00bz", "10.218.11.2"}, {"team12", "VoidHack", "10.218.12.2"},
		{"team13", "Новосибирский Д'Артаньян", "10.218.13.2"}, {"team14", "Trash Querty", "10.218.14.2"},
		{"team15", "Life (Guest)", "10.218.15.2"}, {"team16", "HawkSquad", "10.218.16.2"},
		{"team18", "NeosFun", "10.218.18.2"},
	}
	for _, r := range sibir2018Roster {
		descr := fmt.Sprintf("SibirCTF 2018 roster · ctf01d_id: %s · IP: %s · Активна: yes", r.cfg, r.ip)
		if _, err := ensureTeam(r.name, descr); err != nil {
			return err
		}
		addMeta("SibirCTF 2018", r.name, r.cfg, r.ip)
	}

	sibir2015Names := []string{
		"SuSlo.PAS", "Failers", "FTS", "Life", "Mustang", "OMAVIAT", "Sharlike",
		"SibirTSU", "Zanyato", "TIO", "Luck3rz", "Shikata ga nai", "Hell ZIP", "n57u n00bz",
	}
	for _, name := range sibir2015Names {
		if _, err := ensureTeam(name, "SibirCTF 2015 roster"); err != nil {
			return err
		}
	}

	sibir2014Names := []string{"h34dump", "Yozik", "Brizz", "Mustang", "Сборная АлтГТУ", "Life"}
	for _, name := range sibir2014Names {
		if _, err := ensureTeam(name, "SibirCTF 2014 roster"); err != nil {
			return err
		}
	}

	// keva is the TUSUR team that organizes the SibirCTF / CyberSibir editions
	// (every seeded game lists "keva" as its organizer).
	kevaTeam, err := ensureTeam("keva", "Команда ТУСУР, организатор SibirCTF / CyberSibir")
	if err != nil {
		return err
	}
	// Bundled logo, served from web/public/img/team-logos. Only replaces the
	// generated placeholder so a manually set avatar is preserved.
	kevaLogo := "/img/team-logos/keva.png"
	if shouldBackfillAvatar(kevaTeam.AvatarUrl) {
		updated, err := q.UpdateTeam(ctx, db.UpdateTeamParams{ID: kevaTeam.ID, Name: kevaTeam.Name, AvatarUrl: &kevaLogo})
		if err != nil {
			return fmt.Errorf("keva avatar: %w", err)
		}
		teamByName["keva"] = updated
	}

	// Bundled team logos are applied after the scoreboards (see applyTeamLogos
	// below) so teams that exist only in a scoreboard also get their logo.

	// --- University bindings (manual corrections from the legacy seed) ---------
	bind := func(teamName, uniName, tag, website string) error {
		t, ok := teamByName[teamName]
		if !ok {
			return nil
		}
		uni, ok := uniByName[uniName]
		if !ok {
			return nil
		}
		descr := ptrStr(t.Description)
		if !strings.Contains(descr, tag) {
			if descr != "" {
				descr += " · "
			}
			descr += tag
		}
		uid := uni.ID
		params := db.UpdateTeamParams{ID: t.ID, Name: t.Name, Description: &descr, UniversityID: &uid}
		if website != "" && (t.Website == nil || *t.Website == "") {
			params.Website = &website
		}
		updated, err := q.UpdateTeam(ctx, params)
		if err != nil {
			return fmt.Errorf("binding team %q to university: %w", teamName, err)
		}
		teamByName[teamName] = updated
		return nil
	}
	if err := bind("SharLike", uniAltSTU, "Academic team AltSTU", ""); err != nil {
		return err
	}
	if err := bind("SiBears", uniTSU, "Academic team TSU", ""); err != nil {
		return err
	}
	if err := bind("d34dl1n3", uniSSUGT, "Academic team SSUGT", "https://sgugit.ru"); err != nil {
		return err
	}
	if err := bind("QarabagTeam", uniNSTU, "Academic team NSTU", "https://nstu.ru/"); err != nil {
		return err
	}
	if err := bind("SuSlo.PAS", uniNSU, "Academic team Novosibirsk State University (NSU)", ""); err != nil {
		return err
	}
	// CyberSibir 2026 podium: university + city affiliations.
	if err := bind("CUT", uniCentral, "Центральный университет · г. Москва", ""); err != nil {
		return err
	}
	if err := bind("N0N@me13", uniFSO, "Академия ФСО России · г. Орёл", ""); err != nil {
		return err
	}
	if err := bind("W@zz4bi", uniNSTU, "НГТУ · г. Новосибирск", ""); err != nil {
		return err
	}
	// University affiliations for teams that played our editions (from the
	// AltayCTF participant list). Only teams already present in a roster or
	// scoreboard are bound; non-participants are intentionally not seeded.
	altayBindings := []struct{ team, uni, tag, site string }{
		{"W@zz4b1", uniNSTU, "НГТУ · г. Новосибирск", ""},
		{"NetOverkill", uniKuzSTU, "КузГТУ · г. Кемерово", "https://kuzstu.ru/"},
		{"Error Yager", uniAltSU, "АлтГУ · г. Барнаул", "https://www.asu.ru/"},
	}
	for _, b := range altayBindings {
		if err := bind(b.team, b.uni, b.tag, b.site); err != nil {
			return err
		}
	}
	if err := bind("Mustang", uniTUSUR, "Academic team TUSUR", "https://tusur.ru/"); err != nil {
		return err
	}
	if err := bind("keva", uniTUSUR, "Academic team TUSUR", "https://tusur.ru/"); err != nil {
		return err
	}
	if err := bind("4Ray", uniMIREA, "CTFtime academic team MIREA", "https://www.mirea.ru/"); err != nil {
		return err
	}
	if err := bind("NFB", uniOmSTU, "CTFtime academic team OmSTU", "https://omgtu.ru/"); err != nil {
		return err
	}
	if err := bind("Omaviat", uniOmskAT, "CTFtime academic team Omsk Aviation College", "http://www.oatctf.ru/"); err != nil {
		return err
	}
	if err := bind("paperwhale", uniSibSU, "CTFtime academic team SibSU", "http://sibsau.ru/"); err != nil {
		return err
	}
	if err := bind("UkVQ", uniTUSUR, "CTFtime academic team TUSUR", "https://tusur.ru/"); err != nil {
		return err
	}

	// --- Games (editions by year) ---------------------------------------------
	utc := func(y int, mo time.Month, d, h, mi int) time.Time {
		return time.Date(y, mo, d, h, mi, 0, 0, time.UTC)
	}
	type gameSeed struct {
		name, site, ctftime, logo string
		start, end                time.Time
	}
	gamesData := []gameSeed{
		{name: "SibirCTF 2014", site: "https://sibirctf.org/", logo: "/img/game-logos/sibir-2014.jpg", start: utc(2014, 4, 19, 6, 0), end: utc(2014, 4, 19, 14, 0)},
		{name: "SibirCTF 2015", site: "https://sibirctf.org/", logo: "/img/game-logos/sibir-2015.jpg", start: utc(2015, 4, 18, 6, 0), end: utc(2015, 4, 18, 14, 0)},
		{name: "SibirCTF 2016", site: "https://sibirctf.org/", ctftime: "https://ctftime.org/event/362/", logo: "/img/game-logos/sibir-2016.png", start: utc(2016, 4, 23, 6, 0), end: utc(2016, 4, 23, 14, 0)},
		{name: "SibirCTF 2018", site: "https://sibirctf.org/", start: utc(2018, 10, 21, 4, 0), end: utc(2018, 10, 21, 12, 30)},
		{name: "SibirCTF 2019", site: "https://sibirctf.org/", ctftime: "https://ctftime.org/event/889/", logo: "/img/game-logos/sibir-2019.png", start: utc(2019, 11, 1, 2, 0), end: utc(2019, 11, 1, 12, 0)},
		{name: "SibirCTF 2023", site: "https://vk.com/sibirctf", ctftime: "https://ctftime.org/event/2132/", logo: "/img/game-logos/sibir-2023.jpg", start: utc(2023, 11, 19, 5, 45), end: utc(2023, 11, 19, 13, 0)},
		{name: "CyberSibir 2025", site: "https://vk.com/sibirctf", ctftime: "https://ctftime.org/event/2742/", logo: "/img/game-logos/cybersibir-2025.png", start: utc(2025, 3, 28, 4, 20), end: utc(2025, 3, 28, 12, 20)},
		{name: "CyberSibir 2026", site: "https://vk.com/sibirctf", logo: "/img/game-logos/cyber-2026.jpg", start: utc(2026, 6, 9, 4, 30), end: utc(2026, 6, 9, 12, 30)},
	}
	gameByName := map[string]db.Game{}
	games := make([]db.Game, 0, len(gamesData))
	for _, gd := range gamesData {
		avatar := gd.logo
		if avatar == "" {
			avatar = svgAvatar(gd.name, color())
		}
		g, created, err := getOrCreateGame(ctx, q, gd.name, "keva", gd.start, gd.end, gd.site, gd.ctftime, avatar)
		if err != nil {
			return fmt.Errorf("game %q: %w", gd.name, err)
		}
		logSeed(log, "game", ptrStr(g.Name), created)
		gameByName[gd.name] = g
		games = append(games, g)
	}

	// --- Real CTFtime scoreboards -> results ----------------------------------
	addResult := func(gameName, teamName string, score int32) error {
		g := gameByName[gameName]
		teamName = canonTeam(teamName)
		t, ok := teamByName[teamName]
		if !ok {
			nt, err := ensureTeam(teamName, "Импортировано из scoreboard "+gameName)
			if err != nil {
				return err
			}
			t = nt
		}
		existing, err := q.ListResultsByGameAndTeam(ctx, db.ListResultsByGameAndTeamParams{GameID: g.ID, TeamID: t.ID})
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return nil
		}
		_, err = q.CreateResult(ctx, db.CreateResultParams{GameID: g.ID, TeamID: t.ID, Score: &score})
		return err
	}

	type sbRow struct {
		name   string
		points float64
	}

	scoreboard2025 := []sbRow{
		{"TyumGUard", 8746.5}, {"smiley-from-telega", 8404.6}, {"W@zz4b1", 8145.0},
		{"QarabagTeam", 4646.7}, {"химозный рулет", 4595.1}, {"datapoison", 4578.5},
		{"4Ray", 4476.9}, {"SiBears", 4407.6}, {"N0N@me13", 4350.8}, {"CUT", 4318.1},
		{"Ibeee", 3958.7}, {"vim>nano", 3224.9}, {"NFB", 3190.0}, {"o1d_bu7_go1d", 2891.9},
		{"Циферки", 2693.1}, {"ScareCrow", 2567.4}, {"d34dl1n3", 2451.2}, {"SharLike", 1826.1},
		{"R3T4RD0Z", 1518.7}, {"Netoverkill", 1498.4}, {"kekw", 1442.8}, {"CyberCringe", 1390.8},
		{"CyberPatriots", 1223.8}, {"XAKCET", 810.4},
	}
	for _, r := range scoreboard2025 {
		if err := addResult("CyberSibir 2025", r.name, int32(r.points*1000)); err != nil {
			return err
		}
	}

	// CyberSibir 2026 final scoreboard (jury results, see 2026-cybersibir-jury).
	scoreboard2026 := []sbRow{
		{"CUT", 1155889}, {"N0N@me13", 954308}, {"W@zz4bi", 923699}, {"SiBears", 890034},
		{"1sk4nd3r", 878913}, {"ыыыыЫЫЫЫЫ", 676518}, {"QarabagTeam", 605961}, {"NetOverkill", 564279},
		{"КиберПатриоты", 513650}, {"SharLike", 510505}, {"Циферки", 467274}, {"ufoufo", 428504},
		{"Netrunners", 416576}, {"The Power of Elijah", 353371}, {"d34dl1n3", 283019}, {"4Ray", 282953},
		{"Mustang", 173919}, {"Error Yager", 161678}, {"КиберМамонты", 69414}, {"...", 40230},
		{"avek", 20697},
	}
	for _, r := range scoreboard2026 {
		if err := addResult("CyberSibir 2026", r.name, int32(r.points)); err != nil {
			return err
		}
	}

	scoreboard2023 := []sbRow{
		{"SiBears", 7893.1}, {"ыыыыЫЫЫЫЫ", 7386.4}, {"CubaLibre", 5832.6}, {"QarabagTeam", 5570.8},
		{"Продам гараж за флаги", 5511.9}, {"o1d_bu7_go1d", 4528.1}, {"SharLike", 2817.4},
		{"d34dl1n3", 2658.4}, {"A4PT Reshetneva", 2569.5}, {"ИнфоБесы", 1783.5}, {"LCD", 897.7},
	}
	for _, r := range scoreboard2023 {
		if err := addResult("SibirCTF 2023", r.name, int32(r.points*1000)); err != nil {
			return err
		}
	}

	scoreboard2019 := []sbRow{
		{"Суслобатя", 89430.9}, {"Dragon Hat", 88713.5}, {"Tanuki squad", 55170.2}, {"SiBears", 51399.1},
		{"Omaviat", 47788.8}, {"SharNear", 42291.8}, {"rwx", 41812.4}, {"UkVQ", 36416.3},
		{"4ерниkа", 32676.4}, {"Keva19", 32497.7}, {"a-cool-team", 26495.9}, {"Life", 26139.0},
		{"d34dl1n3", 24739.2}, {"CatchFM", 24347.0}, {"BANOЧKA", 6261.2}, {"Team 16", 4727.0},
	}
	for _, r := range scoreboard2019 {
		if err := addResult("SibirCTF 2019", r.name, int32(r.points)); err != nil {
			return err
		}
	}

	scoreboard2016 := []sbRow{
		{"SiBears", 3250.770}, {"Yozik", 18.790}, {"Team Information Offensive", 45.940},
		{"FoXXXeS", 304.700}, {"Mu574n9", 359.700}, {"!2day", 982.820}, {"SharLike", 1186.730},
		{"(_xXx_-=HOBOCu6uPCKuE_IICbl_1337=-_xXx_)", 1436.590}, {"Life", 1753.340},
		{"xXx_Я_не_ХЛЕБ_я_КОТ_хХх", 1763.670}, {"paperwhale", 0.810},
	}
	for _, r := range scoreboard2016 {
		if err := addResult("SibirCTF 2016", r.name, int32(r.points*1000)); err != nil {
			return err
		}
	}

	// SibirCTF 2015: order from the article; descending synthetic scores keep rank.
	scoreboard2015 := []string{
		"SuSlo.PAS", "Failers", "FTS", "Life", "Mustang", "OMAVIAT", "Sharlike",
		"SibirTSU", "Zanyato", "TIO", "Luck3rz", "Shikata ga nai", "Hell ZIP", "n57u n00bz",
	}
	for idx, name := range scoreboard2015 {
		score := 14000 - 600*idx
		if score < 600 {
			score = 600
		}
		if err := addResult("SibirCTF 2015", name, int32(score)); err != nil {
			return err
		}
	}

	scoreboard2018 := []sbRow{
		{"Новосиб", 7760.15}, {"SharLike", 4450.17}, {"VoidHack", 4028.91}, {"SiBears", 3602.50},
		{"Novosibir", 1736.33}, {"HawkSqu", 1086.32}, {"Void*", 1130.22}, {"RWX", 1068.26},
		{"NeosFun", 1047.88}, {"Life (Guest)", 932.49}, {"CatchFM", 903.86}, {"paperwhale", 890.19},
		{"d34dl1n3", 829.86}, {"Omaviat", 780.55}, {"Life", 778.84}, {"Trash Querty", 618.19},
		{"n57u n00bz", 390.13},
	}
	alias2018 := map[string]string{
		"новосиб": "Новосибирский Д'Артаньян", "novosibir": "Novosibirsk SU X",
		"hawksqu": "HawkSquad", "life (guest)": "Life (Guest)", "trash querty": "Trash Querty",
	}
	for _, r := range scoreboard2018 {
		name := r.name
		if alias, ok := alias2018[strings.ToLower(r.name)]; ok {
			name = alias
		}
		if err := addResult("SibirCTF 2018", name, int32(r.points)); err != nil {
			return err
		}
	}

	scoreboard2014 := []sbRow{
		{"h34dump", 1501}, {"Yozik", 1163}, {"Brizz", 659}, {"Mustang", 626},
		{"Сборная АлтГТУ", 476}, {"Life", 318},
	}
	for _, r := range scoreboard2014 {
		if err := addResult("SibirCTF 2014", r.name, int32(r.points)); err != nil {
			return err
		}
	}
	// UkVQ is imported from the SibirCTF 2019 scoreboard above, so its
	// CTFtime academic binding has to run after scoreboard teams are ensured.
	if err := bind("UkVQ", uniTUSUR, "CTFtime academic team TUSUR", "https://tusur.ru/"); err != nil {
		return err
	}

	// Bundled team logos (served from web/public/img/team-logos) are the canonical
	// source and take precedence over generated placeholders and external
	// (CTFtime) avatars. Runs after the scoreboards so scoreboard-only teams are
	// included too.
	teamLogos := map[string]string{
		"W@zz4bi":            "/img/team-logos/wazz4bi.jpg",
		"W@zz4b1":            "/img/team-logos/wazz4b1.jpg",
		"4Ray":               "/img/team-logos/4Ray.png",
		"Циферки":            "/img/team-logos/ciferki.jpg",
		"CUT":                "/img/team-logos/cut.jpg",
		"CyberCringe":        "/img/team-logos/CyberCringe.jpg",
		"CyberPatriots":      "/img/team-logos/CyberPatriots.jpg",
		"d34dl1n3":           "/img/team-logos/d34dl1n3.png",
		"datapoison":         "/img/team-logos/datapoison.png",
		"химозный рулет":     "/img/team-logos/himozrulet.jpg",
		"kekw":               "/img/team-logos/kekw.jpg",
		"LCD":                "/img/team-logos/lcd.png",
		"NetOverkill":        "/img/team-logos/NetOverkill.jpg",
		"NFB":                "/img/team-logos/NFB.png",
		"N0N@me13":           "/img/team-logos/noname13.jpg",
		"o1d_bu7_go1d":       "/img/team-logos/o1d_bu7_go1d.jpg",
		"QarabagTeam":        "/img/team-logos/QarabagTeam.png",
		"R3T4RD0Z":           "/img/team-logos/R3T4RD0Z.png",
		"ScareCrow":          "/img/team-logos/scarecrow.jpg",
		"SharLike":           "/img/team-logos/SharLike.jpg",
		"SiBears":            "/img/team-logos/SiBears.jpg",
		"smiley-from-telega": "/img/team-logos/smiley-from-telega.jpg",
		"TyumGUard":          "/img/team-logos/TyumGUard.png",
		"vim>nano":           "/img/team-logos/vimnano.png",
		"XAKCET":             "/img/team-logos/XAKCET.jpg",
	}
	for name, logo := range teamLogos {
		t, ok := teamByName[name]
		if !ok {
			continue
		}
		if t.AvatarUrl != nil && *t.AvatarUrl == logo {
			continue
		}
		logoURL := logo
		updated, err := q.UpdateTeam(ctx, db.UpdateTeamParams{ID: t.ID, Name: t.Name, AvatarUrl: &logoURL})
		if err != nil {
			return fmt.Errorf("team %q logo: %w", name, err)
		}
		teamByName[name] = updated
	}

	// --- GameTeams: game<->team links with rank order and ctf01d metadata ------
	nameByTeamID := map[int64]string{}
	for name, t := range teamByName {
		nameByTeamID[t.ID] = name
	}
	for _, g := range games {
		existing, err := q.ListGameTeamsByGame(ctx, g.ID)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			for _, gt := range existing {
				info := ctftimeTeams[nameByTeamID[gt.TeamID]]
				if info.Academic == "" || (gt.TeamType != nil && *gt.TeamType != "") {
					continue
				}
				teamType := "academic"
				if _, err := q.UpdateGameTeam(ctx, db.UpdateGameTeamParams{ID: gt.ID, TeamType: &teamType}); err != nil {
					return fmt.Errorf("game_team %d CTFtime team_type: %w", gt.ID, err)
				}
			}
			continue
		}
		results, err := q.ListResultsByGame(ctx, g.ID)
		if err != nil {
			return err
		}
		sortResultsDesc(results)
		for idx, r := range results {
			gt := db.CreateGameTeamParams{
				GameID:          g.ID,
				TeamID:          r.TeamID,
				Ctf01dOverrides: json.RawMessage("{}"),
				Order:           int32(idx + 1),
			}
			if meta, ok := rosterMeta[ptrStr(g.Name)][strings.ToLower(nameByTeamID[r.TeamID])]; ok {
				if meta.id != "" {
					id := meta.id
					gt.Ctf01dID = &id
				}
				if meta.ip != "" {
					ip := meta.ip
					gt.IpAddress = &ip
				}
			}
			if info := ctftimeTeams[nameByTeamID[r.TeamID]]; info.Academic != "" {
				teamType := "academic"
				gt.TeamType = &teamType
			}
			if _, err := q.CreateGameTeam(ctx, gt); err != nil {
				return fmt.Errorf("game_team game %d team %d: %w", g.ID, r.TeamID, err)
			}
		}
	}

	// --- Services + per-game assignment ---------------------------------------
	type svcSeed struct {
		name, desc, author, branch, repo, lang, game string
	}
	servicesData := []svcSeed{
		// CyberSibir 2026 (jury config; checker-based, no public repo)
		{"XenON-market", "CyberSibir 2026 service (XenON-market)", "CyberSibir", "", "", "", "CyberSibir 2026"},
		{"MSPD2", "CyberSibir 2026 service (MSPD2)", "CyberSibir", "", "", "", "CyberSibir 2026"},
		{"VaultNotes", "CyberSibir 2026 service (VaultNotes)", "CyberSibir", "", "", "", "CyberSibir 2026"},
		{"CWC", "CyberSibir 2026 service (CWC)", "CyberSibir", "", "", "", "CyberSibir 2026"},
		{"IncidentHub", "CyberSibir 2026 service (IncidentHub)", "CyberSibir", "", "", "", "CyberSibir 2026"},
		// CyberSibir 2025
		{"EyeSee", "Service with vulnerabilities for CyberSibir 2025", "CyberSibir", "main", "2025-cybersibir-service-eyesee", "Python", "CyberSibir 2025"},
		{"MSPD", "Service with vulnerabilities for CyberSibir 2025", "CyberSibir", "main", "2025-cybersibir-service-mspd", "Go", "CyberSibir 2025"},
		{"NcDEx", "Service with vulnerabilities for CyberSibir 2025", "CyberSibir", "main", "2025-cybersibir-service-ncdex", "Crystal", "CyberSibir 2025"},
		{"Unpleasant", "Service with vulnerabilities for CyberSibir 2025", "CyberSibir", "master", "2025-cybersibir-service-unpleasant", "HTML", "CyberSibir 2025"},
		{"WrNum", "Service with vulnerabilities for CyberSibir 2025", "CyberSibir", "main", "2025-cybersibir-service-wrnum", "Python", "CyberSibir 2025"},
		{"CyberBank", "Service with vulnerabilities for CyberSibir 2025", "CyberSibir", "main", "2025-cybersibir-service-bank", "JavaScript", "CyberSibir 2025"},
		{"NeuroLink234", "Service with vulnerabilities for CyberSibir 2025", "CyberSibir", "main", "2025-cybersibir-service-neLi234", "C", "CyberSibir 2025"},
		{"BioGuard", "Service with vulnerabilities for CyberSibir 2025", "CyberSibir", "main", "2025-cybersibir-service-BioGuard", "Python", "CyberSibir 2025"},
		// SibirCTF 2023
		{"StickMarket", "SibirCTF 2023 service (StickMarket)", "SibirCTF", "main", "2023-service-sibirctf-stickmarket", "CSS", "SibirCTF 2023"},
		{"SouthParkChat", "SibirCTF 2023 service (SouthParkChat)", "SibirCTF", "main", "2023-service-sibirctf-southparkchat", "Go", "SibirCTF 2023"},
		{"SX", "SibirCTF 2023 service (SX)", "SibirCTF", "main", "2023-service-sibirctf-sx", "Python", "SibirCTF 2023"},
		{"Chef", "SibirCTF 2023 service (Chef)", "SibirCTF", "main", "2023-service-sibirctf-chef", "C", "SibirCTF 2023"},
		{"Card Vault", "SibirCTF 2023 CardVault service", "SibirCTF", "main", "2023-service-sibirctf-cardvault", "Elixir", "SibirCTF 2023"},
		// SibirCTF 2018
		{"maxigram", "SibirCTF 2018 service (maxigram)", "SibirCTF", "master", "2018-service-maxigram", "Python", "SibirCTF 2018"},
		{"The Fakebook", "SibirCTF 2018 service (The Fakebook)", "SibirCTF", "master", "2018-service-thefakebook", "HTML", "SibirCTF 2018"},
		{"The Hole", "SibirCTF 2018 service (The Hole)", "SibirCTF", "master", "2018-service-the-hole", "C++", "SibirCTF 2018"},
		{"Mirai", "SibirCTF 2018 service (Mirai)", "SibirCTF", "master", "2018-service-mirai", "PHP", "SibirCTF 2018"},
		{"LNKS", "SibirCTF 2018 service (LNKS)", "SibirCTF", "master", "2018-service-lnks", "PHP", "SibirCTF 2018"},
		{"Lie2Me", "SibirCTF 2018 service (Lie2Me)", "SibirCTF", "master", "2018-service-lie-to-me", "Perl", "SibirCTF 2018"},
		// SibirCTF 2015
		{"CryChat", "SibirCTF 2015 service (CryChat)", "SibirCTF", "master", "2015-crychat", "PHP", "SibirCTF 2015"},
		{"O'Foody", "SibirCTF 2015 service (O’Foody)", "SibirCTF", "master", "2015-ofoody", "Perl", "SibirCTF 2015"},
		{"CTFGram", "SibirCTF 2015 service (CTFGram)", "SibirCTF", "master", "2015-ctfgram", "JavaScript", "SibirCTF 2015"},
		{"EasyAs", "SibirCTF 2015 service (EasyAs)", "SibirCTF", "master", "2015-easyas", "Python", "SibirCTF 2015"},
	}

	// Pre-load existing game<->service links for idempotency.
	linkedByGame := map[int64]map[int64]bool{}
	for _, g := range games {
		ids, err := q.ListServicesByGame(ctx, g.ID)
		if err != nil {
			return err
		}
		set := map[int64]bool{}
		for _, id := range ids {
			set[id] = true
		}
		linkedByGame[g.ID] = set
	}

	for _, sd := range servicesData {
		archive := ""
		if sd.repo != "" {
			archive = fmt.Sprintf("https://github.com/SibirCTF/%s/archive/refs/heads/%s.zip", sd.repo, sd.branch)
		}
		training := json.RawMessage("{}")
		if sd.lang != "" {
			training = json.RawMessage(fmt.Sprintf(`{"language":%q}`, sd.lang))
		}
		svc, created, err := getOrCreateService(ctx, q, sd.name, sd.desc, sd.author, archive, training, svgAvatar(sd.name, color()))
		if err != nil {
			return fmt.Errorf("service %q: %w", sd.name, err)
		}
		logSeed(log, "service", svc.Name, created)

		g, ok := gameByName[sd.game]
		if !ok {
			continue
		}
		if linkedByGame[g.ID][svc.ID] {
			continue
		}
		if err := q.AddService(ctx, db.AddServiceParams{GameID: g.ID, ServiceID: svc.ID}); err != nil {
			return fmt.Errorf("linking service %q to game %q: %w", sd.name, sd.game, err)
		}
		linkedByGame[g.ID][svc.ID] = true
	}

	// --- Final standings for finished games -----------------------------------
	for _, g := range games {
		if !g.EndsAt.Time.Before(now) {
			continue
		}
		finals, err := q.ListFinalResultsByGame(ctx, g.ID)
		if err != nil {
			return err
		}
		if len(finals) > 0 {
			continue
		}
		results, err := q.ListResultsByGame(ctx, g.ID)
		if err != nil {
			return err
		}
		sortResultsDesc(results)
		for i, r := range results {
			score := int32(0)
			if r.Score != nil {
				score = *r.Score
			}
			pos := int32(i + 1)
			if _, err := q.InsertFinalResult(ctx, db.InsertFinalResultParams{
				GameID: g.ID, TeamID: r.TeamID, Score: score, Position: &pos,
			}); err != nil {
				return fmt.Errorf("final result game %d team %d: %w", g.ID, r.TeamID, err)
			}
		}
		if _, err := q.SetFinalized(ctx, db.SetFinalizedParams{
			ID: g.ID, Finalized: true, FinalizedAt: pgTz(now),
		}); err != nil {
			return fmt.Errorf("finalizing game %d: %w", g.ID, err)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// SibirCTF helpers

func getOrCreateUniversity(ctx context.Context, q *db.Queries, name, siteURL, avatarURL string) (db.University, bool, error) {
	unis, err := q.ListUniversities(ctx, db.ListUniversitiesParams{Limit: 1000, Offset: 0})
	if err != nil {
		return db.University{}, false, err
	}
	for _, u := range unis {
		if u.Name != nil && *u.Name == name {
			return u, false, nil
		}
	}
	var site *string
	if siteURL != "" {
		site = &siteURL
	}
	uni, err := q.CreateUniversity(ctx, db.CreateUniversityParams{Name: &name, SiteUrl: site, AvatarUrl: &avatarURL})
	if err != nil {
		return db.University{}, false, err
	}
	return uni, true, nil
}

func getOrCreateTeam(ctx context.Context, q *db.Queries, name, description, avatarURL string) (db.Team, bool, error) {
	teams, err := q.ListTeams(ctx, db.ListTeamsParams{Limit: 2000, Offset: 0})
	if err != nil {
		return db.Team{}, false, err
	}
	for _, t := range teams {
		if t.Name == name {
			return t, false, nil
		}
	}
	team, err := q.CreateTeam(ctx, db.CreateTeamParams{
		Name:        name,
		Description: &description,
		AvatarUrl:   &avatarURL,
	})
	if err != nil {
		return db.Team{}, false, err
	}
	return team, true, nil
}

func getOrCreateGame(ctx context.Context, q *db.Queries, name, organizer string, startsAt, endsAt time.Time, siteURL, ctftimeURL, avatarURL string) (db.Game, bool, error) {
	games, err := q.ListGames(ctx, db.ListGamesParams{Limit: 1000, Offset: 0})
	if err != nil {
		return db.Game{}, false, err
	}
	for _, g := range games {
		if g.Name != nil && *g.Name == name {
			return g, false, nil
		}
	}
	params := db.CreateGameParams{
		Name:                 &name,
		Organizer:            &organizer,
		StartsAt:             pgTz(startsAt),
		EndsAt:               pgTz(endsAt),
		AvatarUrl:            &avatarURL,
		RegistrationOpensAt:  pgTz(startsAt.Add(-7 * 24 * time.Hour)),
		RegistrationClosesAt: pgTz(startsAt.Add(-1 * 24 * time.Hour)),
		ScoreboardOpensAt:    pgTz(startsAt),
		ScoreboardClosesAt:   pgTz(endsAt.Add(7 * 24 * time.Hour)),
	}
	if siteURL != "" {
		params.SiteUrl = &siteURL
	}
	if ctftimeURL != "" {
		params.CtftimeUrl = &ctftimeURL
	}
	game, err := q.CreateGame(ctx, params)
	if err != nil {
		return db.Game{}, false, err
	}
	return game, true, nil
}

func getOrCreateService(ctx context.Context, q *db.Queries, name, desc, author, archiveURL string, training json.RawMessage, avatarURL string) (db.Service, bool, error) {
	if existing, err := q.GetServiceByName(ctx, name); err == nil {
		return existing, false, nil
	}
	empty := ""
	params := db.CreateServiceParams{
		Name:               name,
		PublicDescription:  &desc,
		PrivateDescription: &empty,
		Author:             &author,
		AvatarUrl:          &avatarURL,
		Public:             true,
		CheckStatus:        "unknown",
		Ctf01dTraining:     training,
	}
	if archiveURL != "" {
		params.ServiceArchiveUrl = &archiveURL
	}
	svc, err := q.CreateService(ctx, params)
	if err != nil {
		return db.Service{}, false, err
	}
	return svc, true, nil
}

func sortResultsDesc(results []db.Result) {
	score := func(r db.Result) int32 {
		if r.Score == nil {
			return 0
		}
		return *r.Score
	}
	// Stable insertion sort: score desc, original (id) order on ties.
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && score(results[j]) > score(results[j-1]); j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

func backfillCTFtimeTeamInfo(ctx context.Context, q *db.Queries, team db.Team, seedDescription string, info ctftimeTeamInfo) (db.Team, error) {
	if info.ID == "" {
		return team, nil
	}

	description := ptrStr(team.Description)
	if description == "" {
		description = seedDescription
	}
	description = enrichTeamDescription(description, info)

	params := db.UpdateTeamParams{ID: team.ID, Name: team.Name}
	if description != ptrStr(team.Description) {
		params.Description = &description
	}
	if info.AvatarURL != "" && shouldBackfillAvatar(team.AvatarUrl) {
		params.AvatarUrl = &info.AvatarURL
	}
	if params.Description == nil && params.AvatarUrl == nil {
		return team, nil
	}

	updated, err := q.UpdateTeam(ctx, params)
	if err != nil {
		return db.Team{}, err
	}
	return updated, nil
}

func enrichTeamDescription(description string, info ctftimeTeamInfo) string {
	if URL := info.profileURL(); URL != "" {
		description = appendDescriptionPart(description, "CTFtime: "+URL)
	}
	if info.Country != "" {
		description = appendDescriptionPart(description, "CTFtime country: "+info.Country)
	}
	return description
}

func appendDescriptionPart(description, part string) string {
	if part == "" || strings.Contains(description, part) {
		return description
	}
	if description == "" {
		return part
	}
	return description + " · " + part
}

func shouldBackfillAvatar(avatarURL *string) bool {
	if avatarURL == nil || *avatarURL == "" {
		return true
	}
	return strings.HasPrefix(*avatarURL, "data:image/svg+xml")
}

func svgAvatar(text, bg string) string {
	initial := "?"
	if r := []rune(text); len(r) > 0 {
		initial = strings.ToUpper(string(r[0]))
	}
	svg := fmt.Sprintf(
		"<svg xmlns='http://www.w3.org/2000/svg' width='96' height='96'>"+
			"<rect width='100%%' height='100%%' fill='%s' />"+
			"<text x='50%%' y='56%%' dominant-baseline='middle' text-anchor='middle' "+
			"font-family='Arial, Helvetica, sans-serif' font-size='48' fill='#fff'>%s</text></svg>",
		bg, initial)
	enc := strings.ReplaceAll(url.QueryEscape(svg), "+", "%20")
	return "data:image/svg+xml;utf8," + enc
}

// -----------------------------------------------------------------------------
// Minimal seed helpers (admin + baseline users)

func seedAdmin(ctx context.Context, q *db.Queries, password string) (db.User, bool, error) {
	existing, err := q.GetUserByUserName(ctx, "admin")
	if err == nil {
		return existing, false, nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return db.User{}, false, fmt.Errorf("hashing password: %w", err)
	}

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		UserName:       "admin",
		DisplayName:    "Administrator",
		Role:           "admin",
		Rating:         0,
		PasswordDigest: &hash,
	})
	if err != nil {
		return db.User{}, false, err
	}
	return user, true, nil
}

func seedUser(ctx context.Context, q *db.Queries, userName, displayName, role, password string) (db.User, bool, error) {
	existing, err := q.GetUserByUserName(ctx, userName)
	if err == nil {
		return existing, false, nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return db.User{}, false, fmt.Errorf("hashing password: %w", err)
	}

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		UserName:       userName,
		DisplayName:    displayName,
		Role:           role,
		Rating:         0,
		PasswordDigest: &hash,
	})
	if err != nil {
		return db.User{}, false, err
	}
	return user, true, nil
}

func logSeed(log *zap.Logger, entity, name string, created bool) {
	if created {
		log.Info("created", zap.String("entity", entity), zap.String("name", name))
	} else {
		log.Info("already exists", zap.String("entity", entity), zap.String("name", name))
	}
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func pgTz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}
