import { useState, useEffect, useCallback } from "react";
import * as resultsApi from "../api/results";
import type { Result, ResultCreate, ResultUpdate } from "../api/results";
import { formatDateTime } from "../components/DetailInfo";
import { ErrorDisplay, ActionButton } from "../components/ErrorDisplay";
import { usePageTitle } from "../components/usePageTitle";
import { useAuth } from "../auth/AuthContext";
import { useI18n } from "../i18n/I18nContext";

export default function ResultsPage() {
  const { t } = useI18n();
  usePageTitle(t("Results"));
  const { isPlayer } = useAuth();

  const [results, setResults] = useState<Result[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [filterGameId, setFilterGameId] = useState("");
  const [filterTeamId, setFilterTeamId] = useState("");

  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState<ResultCreate>({
    game_id: 0,
    team_id: 0,
    score: 0,
  });
  const [creating, setCreating] = useState(false);

  const [editingId, setEditingId] = useState<number | null>(null);
  const [editForm, setEditForm] = useState<ResultUpdate>({});
  const [saving, setSaving] = useState(false);

  const fetchResults = useCallback(async () => {
    setLoading(true);
    setError(null);
    const query: { game_id?: number; team_id?: number } = {};
    if (filterGameId) query.game_id = Number(filterGameId);
    if (filterTeamId) query.team_id = Number(filterTeamId);
    const { data, error: err } = await resultsApi.listResults(query);
    if (err) {
      setError(err);
    } else if (data) {
      setResults(data.items);
    }
    setLoading(false);
  }, [filterGameId, filterTeamId]);

  useEffect(() => {
    void fetchResults();
  }, [fetchResults]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    const { error: err } = await resultsApi.createResult(createForm);
    setCreating(false);
    if (err) {
      setError(err);
      return;
    }
    setCreateForm({ game_id: 0, team_id: 0, score: 0 });
    setShowCreate(false);
    await fetchResults();
  };

  const startEdit = (r: Result) => {
    setEditingId(r.id);
    setEditForm({ score: r.score ?? undefined });
  };

  const handleSave = async () => {
    if (editingId === null) return;
    setSaving(true);
    const { error: err } = await resultsApi.updateResult(editingId, editForm);
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    setEditingId(null);
    await fetchResults();
  };

  const handleDelete = async (id: number) => {
    const { error: err } = await resultsApi.deleteResult(id);
    if (err) {
      setError(err);
      return;
    }
    await fetchResults();
  };

  return (
    <div className="page">
      <div className="page-header">
        <div className="inline-form">
          <input
            type="number"
            placeholder={t("Filter by Game ID")}
            value={filterGameId}
            onChange={(e) => setFilterGameId(e.target.value)}
          />
          <input
            type="number"
            placeholder={t("Filter by Team ID")}
            value={filterTeamId}
            onChange={(e) => setFilterTeamId(e.target.value)}
          />
          <button className="btn btn-sm" onClick={() => void fetchResults()}>
            {t("Filter")}
          </button>
          {(filterGameId || filterTeamId) && (
            <button
              className="btn btn-sm"
              onClick={() => {
                setFilterGameId("");
                setFilterTeamId("");
              }}
            >
              {t("Clear")}
            </button>
          )}
        </div>
        {isPlayer && (
          <button
            className="btn btn-primary"
            onClick={() => setShowCreate(!showCreate)}
          >
            {showCreate ? t("Cancel") : t("Create Result")}
          </button>
        )}
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="create-form">
          <div className="form-group">
            <label>{t("Game ID")}</label>
            <input
              type="number"
              value={createForm.game_id || ""}
              onChange={(e) =>
                setCreateForm((f) => ({
                  ...f,
                  game_id: Number(e.target.value),
                }))
              }
              required
            />
          </div>
          <div className="form-group">
            <label>{t("Team ID")}</label>
            <input
              type="number"
              value={createForm.team_id || ""}
              onChange={(e) =>
                setCreateForm((f) => ({
                  ...f,
                  team_id: Number(e.target.value),
                }))
              }
              required
            />
          </div>
          <div className="form-group">
            <label>{t("Score")}</label>
            <input
              type="number"
              value={createForm.score ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, score: Number(e.target.value) }))
              }
            />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? t("Creating...") : t("Create")}
          </button>
        </form>
      )}

      <ErrorDisplay error={error} onRetry={fetchResults} />

      {editingId !== null && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void handleSave();
          }}
          className="edit-form"
        >
          <div className="form-group">
            <label>{t("Score")}</label>
            <input
              type="number"
              value={editForm.score ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, score: Number(e.target.value) }))
              }
            />
          </div>
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {saving ? t("Saving...") : t("Save")}
            </button>
            <button
              type="button"
              className="btn"
              onClick={() => setEditingId(null)}
            >
              {t("Cancel")}
            </button>
          </div>
        </form>
      )}

      {loading ? (
        <div className="loading">{t("Loading...")}</div>
      ) : results.length === 0 ? (
        <div className="empty-state">{t("No results found")}</div>
      ) : (
        <div className="table-shell table-shell-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>{t("Game ID")}</th>
                <th>{t("Team ID")}</th>
                <th className="numeric">{t("Score")}</th>
                <th>{t("Created")}</th>
                <th>{t("Updated")}</th>
                {isPlayer && <th>{t("Actions")}</th>}
              </tr>
            </thead>
            <tbody>
              {results.map((r) => (
                <tr key={r.id}>
                  <td>{r.id}</td>
                  <td>{r.game_id}</td>
                  <td>{r.team_id}</td>
                  <td className="numeric score-cell">
                    {r.score?.toLocaleString() ?? "—"}
                  </td>
                  <td>{formatDateTime(r.created_at)}</td>
                  <td>{formatDateTime(r.updated_at)}</td>
                  {isPlayer && (
                    <td>
                      <div className="actions-cell">
                        <ActionButton onClick={() => startEdit(r)}>
                          {t("Edit")}
                        </ActionButton>
                        <ActionButton
                          onClick={() => handleDelete(r.id)}
                          variant="danger"
                          confirm={t("Delete this result?")}
                        >
                          {t("Delete")}
                        </ActionButton>
                      </div>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
