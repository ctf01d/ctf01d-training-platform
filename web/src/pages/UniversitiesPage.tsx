import { useState, useEffect, useCallback } from "react";
import * as universitiesApi from "../api/universities";
import type {
  University,
  UniversityCreate,
  UniversityUpdate,
} from "../api/universities";
import { CardGrid, EntityCard, CardMeta, Pagination } from "../components/Card";
import { ErrorDisplay, ActionButton } from "../components/ErrorDisplay";

export default function UniversitiesPage() {
  const [universities, setUniversities] = useState<University[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 20;

  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState<UniversityCreate>({});
  const [creating, setCreating] = useState(false);

  const [editingId, setEditingId] = useState<number | null>(null);
  const [editForm, setEditForm] = useState<UniversityUpdate>({});
  const [saving, setSaving] = useState(false);

  const fetchUniversities = useCallback(async () => {
    setLoading(true);
    setError(null);
    const { data, error: err } = await universitiesApi.listUniversities({
      page,
      per_page: perPage,
    });
    if (err) {
      setError(err);
    } else if (data) {
      setUniversities(data.items);
      setTotal(data.pagination.total);
    }
    setLoading(false);
  }, [page]);

  useEffect(() => {
    void fetchUniversities();
  }, [fetchUniversities]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    const { error: err } = await universitiesApi.createUniversity(createForm);
    setCreating(false);
    if (err) {
      setError(err);
      return;
    }
    setCreateForm({});
    setShowCreate(false);
    await fetchUniversities();
  };

  const startEdit = (u: University) => {
    setEditingId(u.id);
    setEditForm({
      name: u.name ?? undefined,
      site_url: u.site_url ?? undefined,
      avatar_url: u.avatar_url ?? undefined,
    });
  };

  const handleSave = async () => {
    if (editingId === null) return;
    setSaving(true);
    const { error: err } = await universitiesApi.updateUniversity(
      editingId,
      editForm,
    );
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    setEditingId(null);
    await fetchUniversities();
  };

  const handleDelete = async (id: number) => {
    const { error: err } = await universitiesApi.deleteUniversity(id);
    if (err) {
      setError(err);
      return;
    }
    await fetchUniversities();
  };

  const safeUrl = (u: string | undefined | null) => {
    if (!u) return null;
    try {
      const parsed = new URL(u);
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:")
        return null;
      return u;
    } catch {
      return null;
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Universities</h1>
        <button
          className="btn btn-primary"
          onClick={() => setShowCreate(!showCreate)}
        >
          {showCreate ? "Cancel" : "Create University"}
        </button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="create-form">
          <div className="form-group">
            <label>Name</label>
            <input
              value={createForm.name ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, name: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Site URL</label>
            <input
              value={createForm.site_url ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, site_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Avatar URL</label>
            <input
              value={createForm.avatar_url ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, avatar_url: e.target.value }))
              }
            />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? "Creating..." : "Create"}
          </button>
        </form>
      )}

      <ErrorDisplay error={error} onRetry={fetchUniversities} />

      {editingId !== null && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void handleSave();
          }}
          className="edit-form"
        >
          <div className="form-group">
            <label>Name</label>
            <input
              value={editForm.name ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, name: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Site URL</label>
            <input
              value={editForm.site_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, site_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Avatar URL</label>
            <input
              value={editForm.avatar_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, avatar_url: e.target.value }))
              }
            />
          </div>
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {saving ? "Saving..." : "Save"}
            </button>
            <button
              type="button"
              className="btn"
              onClick={() => setEditingId(null)}
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      <CardGrid
        loading={loading}
        isEmpty={universities.length === 0}
        emptyMessage="No universities found"
      >
        {universities.map((u) => (
          <EntityCard
            key={u.id}
            to={`/universities/${u.id}`}
            avatarUrl={u.avatar_url}
            avatarText={u.name ?? "?"}
            title={u.name ?? "—"}
            footer={
              <>
                <ActionButton onClick={() => startEdit(u)}>Edit</ActionButton>
                <ActionButton
                  onClick={() => handleDelete(u.id)}
                  variant="danger"
                  confirm="Delete this university?"
                >
                  Delete
                </ActionButton>
              </>
            }
          >
            <CardMeta label="Site">
              {safeUrl(u.site_url) ? (
                <a href={u.site_url!} target="_blank" rel="noreferrer">
                  {u.site_url}
                </a>
              ) : (
                (u.site_url ?? "—")
              )}
            </CardMeta>
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
