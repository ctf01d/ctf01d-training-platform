import { useState, useEffect, useCallback } from "react";
import { Link, useNavigate } from "react-router-dom";
import * as gamesApi from "../api/games";
import type { Game, GameCreate } from "../api/games";
import * as teamsApi from "../api/teams";
import { CardGrid, CardBadge, Pagination } from "../components/Card";
import { ErrorDisplay } from "../components/ErrorDisplay";
import { RelativeTime, Duration } from "../components/DetailInfo";
import { usePageTitle } from "../components/usePageTitle";
import { useAuth } from "../auth/AuthContext";
import { useI18n } from "../i18n/I18nContext";
import { datetimeLocalToRFC3339 } from "../api/datetime";
import {
  DEFAULT_GAME_THEME,
  DEFAULT_GAME_REQUIREMENTS,
} from "./gamePlanningTemplate";

/**
 * Map of lower-cased organizer name -> team (or null when no team matches).
 * Misses are recorded as null so we don't keep re-querying them.
 */
type OrganizerTeams = Record<string, { id: number; name: string } | null>;

export default function GamesPage() {
  const { t } = useI18n();
  usePageTitle(t("Games"));
  const { isPlayer } = useAuth();
  const navigate = useNavigate();
  const [games, setGames] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 20;
  const [searchQuery, setSearchQuery] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState<GameCreate>({});
  const [creating, setCreating] = useState(false);
  const [organizerTeams, setOrganizerTeams] = useState<OrganizerTeams>({});
  const [planningGames, setPlanningGames] = useState<Game[]>([]);
  const [teamOptions, setTeamOptions] = useState<
    { id: number; name: string }[]
  >([]);
  // The exact schedule is often unknown when a game is first created, so the
  // form lets the organizer pick just a year instead of a full datetime.
  const [dateMode, setDateMode] = useState<"datetime" | "year">("datetime");
  const [year, setYear] = useState("");

  const fetchGames = useCallback(async () => {
    setLoading(true);
    setError(null);
    const { data, error: err } = await gamesApi.listGames({
      page,
      per_page: perPage,
      q: searchQuery || undefined,
      published: true,
    });
    if (err) {
      setError(err);
    } else if (data) {
      setGames(data.items);
      setTotal(data.pagination.total);
    }
    setLoading(false);
  }, [page, searchQuery]);

  const fetchPlanningGames = useCallback(async () => {
    if (!isPlayer) {
      setPlanningGames([]);
      return;
    }
    const { data } = await gamesApi.listGames({
      per_page: 100,
      published: false,
    });
    if (data) setPlanningGames(data.items);
  }, [isPlayer]);

  useEffect(() => {
    void fetchGames();
  }, [fetchGames]);

  useEffect(() => {
    void fetchPlanningGames();
  }, [fetchPlanningGames]);

  // Load existing teams to offer as organizer suggestions when the create form
  // opens. The organizer is stored as a team name, so matching an existing team
  // lets GamesPage render it as a link.
  useEffect(() => {
    if (!showCreate || teamOptions.length > 0) return;
    void teamsApi
      .listAllTeams()
      .then((teams) =>
        setTeamOptions(teams.map((t) => ({ id: t.id, name: t.name }))),
      );
  }, [showCreate, teamOptions.length]);

  // Resolve organizer names to teams so they can be rendered as links.
  useEffect(() => {
    const names = Array.from(
      new Set(
        [...games, ...planningGames]
          .map((g) => g.organizer?.trim())
          .filter((n): n is string => !!n)
          .map((n) => n.toLowerCase()),
      ),
    ).filter((n) => !(n in organizerTeams));
    if (names.length === 0) return;
    void Promise.all(
      names.map(async (name) => {
        const { data } = await teamsApi.listTeams({ q: name, per_page: 20 });
        const match = data?.items.find((t) => t.name.toLowerCase() === name);
        return [name, match] as const;
      }),
    ).then((pairs) => {
      setOrganizerTeams((prev) => {
        const next = { ...prev };
        for (const [name, match] of pairs) {
          next[name] = match ? { id: match.id, name: match.name } : null;
        }
        return next;
      });
    });
  }, [games, planningGames, organizerTeams]);

  // Build the starts_at/ends_at payload from whichever date mode is active.
  // In "year" mode a bare year is stored as midnight on January 1st so it still
  // fits the date-time schema; ends_at is left unset until the schedule is known.
  const resolveDates = (): Pick<GameCreate, "starts_at" | "ends_at"> => {
    if (dateMode === "year") {
      const y = year.trim();
      return {
        starts_at: /^\d{4}$/.test(y) ? `${y}-01-01T00:00:00Z` : undefined,
        ends_at: undefined,
      };
    }
    return {
      starts_at: datetimeLocalToRFC3339(form.starts_at),
      ends_at: datetimeLocalToRFC3339(form.ends_at),
    };
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    const { data, error: err } = await gamesApi.createGame({
      ...form,
      ...resolveDates(),
    });
    setCreating(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      navigate(`/games/${data.id}`);
    }
  };

  // Create a draft game prefilled with the CyberSibir planning template and
  // open its planning page; it stays out of the games list until published.
  const handlePlan = async () => {
    setCreating(true);
    const { data, error: err } = await gamesApi.createGame({
      ...form,
      ...resolveDates(),
      published: false,
      theme: DEFAULT_GAME_THEME,
      requirements: DEFAULT_GAME_REQUIREMENTS,
    });
    setCreating(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      navigate(`/games/${data.id}/planning`);
    }
  };

  return (
    <div className="page games-page">
      <div className="page-header">
        <div className="filters">
          <input
            placeholder={t("Search games...")}
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        {isPlayer && (
          <button
            className="btn btn-primary"
            onClick={() => setShowCreate(!showCreate)}
          >
            {showCreate ? t("Cancel") : t("Create Game")}
          </button>
        )}
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="create-form">
          <div className="form-group">
            <label>{t("Name")}</label>
            <input
              value={form.name ?? ""}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            />
          </div>
          <div className="form-group">
            <label>{t("Organizer Team")}</label>
            <input
              list="organizer-team-options"
              placeholder={t("Select an existing team")}
              value={form.organizer ?? ""}
              onChange={(e) =>
                setForm((f) => ({ ...f, organizer: e.target.value }))
              }
            />
            <datalist id="organizer-team-options">
              {teamOptions.map((t) => (
                <option key={t.id} value={t.name} />
              ))}
            </datalist>
          </div>
          <div className="form-group">
            <label>{t("Schedule")}</label>
            <select
              value={dateMode}
              onChange={(e) =>
                setDateMode(e.target.value as "datetime" | "year")
              }
            >
              <option value="datetime">{t("Exact date & time")}</option>
              <option value="year">{t("Year only")}</option>
            </select>
          </div>
          {dateMode === "year" ? (
            <div className="form-group">
              <label>{t("Year")}</label>
              <input
                type="number"
                inputMode="numeric"
                min={2000}
                max={2100}
                placeholder="2027"
                value={year}
                onChange={(e) => setYear(e.target.value)}
              />
            </div>
          ) : (
            <>
              <div className="form-group">
                <label>{t("Starts At")}</label>
                <input
                  type="datetime-local"
                  value={form.starts_at ?? ""}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, starts_at: e.target.value }))
                  }
                />
              </div>
              <div className="form-group">
                <label>{t("Ends At")}</label>
                <input
                  type="datetime-local"
                  value={form.ends_at ?? ""}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, ends_at: e.target.value }))
                  }
                />
              </div>
            </>
          )}
          <div className="form-actions">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={creating}
            >
              {creating ? t("Creating...") : t("Create")}
            </button>
            <button
              type="button"
              className="btn"
              disabled={creating}
              onClick={() => void handlePlan()}
              title={t("Create a draft with requirements and open planning")}
            >
              {t("Plan Game")}
            </button>
          </div>
        </form>
      )}

      {isPlayer && planningGames.length > 0 && (
        <section className="games-section">
          <h3 className="games-section-title">{t("In planning")}</h3>
          <CardGrid isEmpty={false}>
            {planningGames.map((g) => (
              <GameCard
                key={g.id}
                game={g}
                organizerTeams={organizerTeams}
                planning
              />
            ))}
          </CardGrid>
        </section>
      )}

      <ErrorDisplay error={error} onRetry={fetchGames} />

      <section className="games-section">
        {isPlayer && planningGames.length > 0 && (
          <h3 className="games-section-title">{t("Published")}</h3>
        )}
        <CardGrid
          loading={loading}
          isEmpty={games.length === 0}
          emptyMessage={t("No games found")}
        >
          {games.map((g) => (
            <GameCard key={g.id} game={g} organizerTeams={organizerTeams} />
          ))}
        </CardGrid>
      </section>

      <Pagination
        page={page}
        perPage={perPage}
        total={total}
        onPageChange={setPage}
      />
    </div>
  );
}

function GameCard({
  game,
  organizerTeams,
  planning = false,
}: {
  game: Game;
  organizerTeams: OrganizerTeams;
  planning?: boolean;
}) {
  const { t } = useI18n();
  const [imageFailed, setImageFailed] = useState(false);
  const title = game.name ?? `${t("Game")} #${game.id}`;
  const hasImage = Boolean(game.avatar_url && !imageFailed);
  const organizerTeam = game.organizer
    ? organizerTeams[game.organizer.trim().toLowerCase()]
    : null;
  // Planning drafts open their planning workspace; published games open detail.
  const to = planning ? `/games/${game.id}/planning` : `/games/${game.id}`;

  return (
    <article className="game-card">
      <div className="game-card-content">
        <div className="game-card-heading">
          <Link to={to} className="game-card-title">
            {title}
          </Link>
          <div className="game-card-badges">
            {planning ? (
              <CardBadge variant="pending">{t("planning")}</CardBadge>
            ) : (
              <CardBadge variant={game.status ?? "unknown"}>
                {t(game.status ?? "unknown")}
              </CardBadge>
            )}
            {game.finalized && (
              <CardBadge variant="approved">{t("finalized")}</CardBadge>
            )}
          </div>
        </div>

        <dl className="game-card-meta">
          <div>
            <dt>{t("Organizer")}</dt>
            <dd>
              {organizerTeam ? (
                <Link to={`/teams/${organizerTeam.id}`}>
                  {organizerTeam.name}
                </Link>
              ) : (
                (game.organizer ?? "—")
              )}
            </dd>
          </div>
          <div>
            <dt>{t("Date")}</dt>
            <dd>
              <RelativeTime value={game.starts_at} />
            </dd>
          </div>
          <div>
            <dt>{t("Duration")}</dt>
            <dd>
              <Duration start={game.starts_at} end={game.ends_at} />
            </dd>
          </div>
          <div>
            <dt>{t("Registration")}</dt>
            <dd>
              <CardBadge variant={game.registration_status ?? "unscheduled"}>
                {t(game.registration_status ?? "unscheduled")}
              </CardBadge>
            </dd>
          </div>
          <div>
            <dt>{t("Scoreboard")}</dt>
            <dd>
              <CardBadge variant={game.scoreboard_status ?? "closed"}>
                {t(game.scoreboard_status ?? "closed")}
              </CardBadge>
            </dd>
          </div>
        </dl>
      </div>

      <Link
        to={to}
        className="game-card-media"
        tabIndex={-1}
        aria-hidden="true"
      >
        {hasImage ? (
          <img
            src={game.avatar_url ?? ""}
            alt=""
            loading="lazy"
            onError={() => setImageFailed(true)}
          />
        ) : (
          <span>{title.trim().charAt(0).toUpperCase() || "?"}</span>
        )}
      </Link>
    </article>
  );
}
