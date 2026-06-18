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

/**
 * Map of lower-cased organizer name -> team (or null when no team matches).
 * Misses are recorded as null so we don't keep re-querying them.
 */
type OrganizerTeams = Record<string, { id: number; name: string } | null>;

export default function GamesPage() {
  usePageTitle("Games");
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

  const fetchGames = useCallback(async () => {
    setLoading(true);
    setError(null);
    const { data, error: err } = await gamesApi.listGames({
      page,
      per_page: perPage,
      q: searchQuery || undefined,
    });
    if (err) {
      setError(err);
    } else if (data) {
      setGames(data.items);
      setTotal(data.pagination.total);
    }
    setLoading(false);
  }, [page, searchQuery]);

  useEffect(() => {
    void fetchGames();
  }, [fetchGames]);

  // Resolve organizer names to teams so they can be rendered as links.
  useEffect(() => {
    const names = Array.from(
      new Set(
        games
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
  }, [games, organizerTeams]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    const { data, error: err } = await gamesApi.createGame(form);
    setCreating(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      navigate(`/games/${data.id}`);
    }
  };

  return (
    <div className="page games-page">
      <div className="page-header">
        <div className="filters">
          <input
            placeholder="Search games..."
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
            {showCreate ? "Cancel" : "Create Game"}
          </button>
        )}
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="create-form">
          <div className="form-group">
            <label>Name</label>
            <input
              value={form.name ?? ""}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            />
          </div>
          <div className="form-group">
            <label>Organizer</label>
            <input
              value={form.organizer ?? ""}
              onChange={(e) =>
                setForm((f) => ({ ...f, organizer: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Starts At</label>
            <input
              type="datetime-local"
              value={form.starts_at ?? ""}
              onChange={(e) =>
                setForm((f) => ({ ...f, starts_at: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Ends At</label>
            <input
              type="datetime-local"
              value={form.ends_at ?? ""}
              onChange={(e) =>
                setForm((f) => ({ ...f, ends_at: e.target.value }))
              }
            />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? "Creating..." : "Create"}
          </button>
        </form>
      )}

      <ErrorDisplay error={error} onRetry={fetchGames} />

      <CardGrid
        loading={loading}
        isEmpty={games.length === 0}
        emptyMessage="No games found"
      >
        {games.map((g) => (
          <GameCard key={g.id} game={g} organizerTeams={organizerTeams} />
        ))}
      </CardGrid>

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
}: {
  game: Game;
  organizerTeams: OrganizerTeams;
}) {
  const [imageFailed, setImageFailed] = useState(false);
  const title = game.name ?? `Game #${game.id}`;
  const hasImage = Boolean(game.avatar_url && !imageFailed);
  const organizerTeam = game.organizer
    ? organizerTeams[game.organizer.trim().toLowerCase()]
    : null;

  return (
    <article className="game-card">
      <div className="game-card-content">
        <div className="game-card-heading">
          <Link to={`/games/${game.id}`} className="game-card-title">
            {title}
          </Link>
          <div className="game-card-badges">
            <CardBadge variant={game.status ?? "unknown"}>
              {game.status ?? "unknown"}
            </CardBadge>
            {game.finalized && (
              <CardBadge variant="approved">finalized</CardBadge>
            )}
          </div>
        </div>

        <dl className="game-card-meta">
          <div>
            <dt>Organizer</dt>
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
            <dt>Date</dt>
            <dd>
              <RelativeTime value={game.starts_at} />
            </dd>
          </div>
          <div>
            <dt>Duration</dt>
            <dd>
              <Duration start={game.starts_at} end={game.ends_at} />
            </dd>
          </div>
          <div>
            <dt>Registration</dt>
            <dd>
              <CardBadge variant={game.registration_status ?? "unscheduled"}>
                {game.registration_status ?? "unscheduled"}
              </CardBadge>
            </dd>
          </div>
          <div>
            <dt>Scoreboard</dt>
            <dd>
              <CardBadge variant={game.scoreboard_status ?? "closed"}>
                {game.scoreboard_status ?? "closed"}
              </CardBadge>
            </dd>
          </div>
        </dl>
      </div>

      <Link
        to={`/games/${game.id}`}
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
