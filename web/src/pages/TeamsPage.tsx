import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import * as teamsApi from "../api/teams";
import type { Team, TeamCreate } from "../api/teams";
import { CardGrid, EntityCard, CardMeta, Pagination } from "../components/Card";
import { ErrorDisplay } from "../components/ErrorDisplay";
import { usePageTitle } from "../components/usePageTitle";
import { useAuth } from "../auth/AuthContext";

export default function TeamsPage() {
  usePageTitle("Teams");
  const { isPlayer } = useAuth();
  const navigate = useNavigate();
  const [teams, setTeams] = useState<Team[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 20;
  const [searchQuery, setSearchQuery] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState<TeamCreate>({ name: "" });
  const [creating, setCreating] = useState(false);
  const [memberCounts, setMemberCounts] = useState<Record<number, number>>({});

  const fetchTeams = useCallback(async () => {
    setLoading(true);
    setError(null);
    const { data, error: err } = await teamsApi.listTeams({
      page,
      per_page: perPage,
      q: searchQuery || undefined,
    });
    if (err) {
      setError(err);
    } else if (data) {
      setTeams(data.items);
      setTotal(data.pagination.total);
    }
    setLoading(false);
  }, [page, searchQuery]);

  useEffect(() => {
    void fetchTeams();
  }, [fetchTeams]);

  useEffect(() => {
    const counts: Record<number, number> = {};
    let pending = teams.length;
    if (pending === 0) return;
    for (const team of teams) {
      void teamsApi.listTeamMembers(team.id).then(({ data }) => {
        counts[team.id] = data?.items.length ?? 0;
        pending--;
        if (pending === 0) setMemberCounts((prev) => ({ ...prev, ...counts }));
      });
    }
  }, [teams]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    const { data, error: err } = await teamsApi.createTeam(form);
    setCreating(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      navigate(`/teams/${data.id}`);
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <div className="filters">
          <input
            placeholder="Search teams..."
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
            {showCreate ? "Cancel" : "Create Team"}
          </button>
        )}
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="create-form">
          <div className="form-group">
            <label>Name</label>
            <input
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              required
            />
          </div>
          <div className="form-group">
            <label>Description</label>
            <input
              value={form.description ?? ""}
              onChange={(e) =>
                setForm((f) => ({ ...f, description: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Website</label>
            <input
              value={form.website ?? ""}
              onChange={(e) =>
                setForm((f) => ({ ...f, website: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Avatar URL</label>
            <input
              value={form.avatar_url ?? ""}
              onChange={(e) =>
                setForm((f) => ({ ...f, avatar_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>University ID</label>
            <input
              type="number"
              value={form.university_id ?? ""}
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  university_id: e.target.value ? Number(e.target.value) : null,
                }))
              }
            />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? "Creating..." : "Create"}
          </button>
        </form>
      )}

      <ErrorDisplay error={error} onRetry={fetchTeams} />

      <CardGrid
        loading={loading}
        isEmpty={teams.length === 0}
        emptyMessage="No teams found"
      >
        {teams.map((t) => (
          <EntityCard
            key={t.id}
            to={`/teams/${t.id}`}
            avatarUrl={t.avatar_url}
            avatarText={t.name}
            title={t.name}
          >
            {t.description && (
              <CardMeta label="About">{t.description}</CardMeta>
            )}
            <CardMeta label="Members">{memberCounts[t.id] ?? "..."}</CardMeta>
            {t.website && (
              <CardMeta label="Website">
                <a
                  href={t.website}
                  target="_blank"
                  rel="noreferrer"
                  onClick={(e) => e.stopPropagation()}
                >
                  {t.website}
                </a>
              </CardMeta>
            )}
          </EntityCard>
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
