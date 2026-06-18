import { useState, useEffect, useCallback } from "react";
import { Link } from "react-router-dom";
import * as universitiesApi from "../api/universities";
import type { University, UniversityCreate } from "../api/universities";
import { CardGrid, Pagination } from "../components/Card";
import { ErrorDisplay } from "../components/ErrorDisplay";

export default function UniversitiesPage() {
  const [universities, setUniversities] = useState<University[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 20;
  const [searchQuery, setSearchQuery] = useState("");

  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState<UniversityCreate>({});
  const [creating, setCreating] = useState(false);

  const fetchUniversities = useCallback(async () => {
    setLoading(true);
    setError(null);
    const { data, error: err } = await universitiesApi.listUniversities({
      page,
      per_page: perPage,
      q: searchQuery || undefined,
    });
    if (err) {
      setError(err);
    } else if (data) {
      setUniversities(data.items);
      setTotal(data.pagination.total);
    }
    setLoading(false);
  }, [page, searchQuery]);

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

  return (
    <div className="page universities-page">
      <div className="page-header">
        <h1>Universities</h1>
        <button
          className="btn btn-primary"
          onClick={() => setShowCreate(!showCreate)}
        >
          {showCreate ? "Cancel" : "Create University"}
        </button>
      </div>

      <div className="filters">
        <input
          placeholder="Search universities..."
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            setPage(1);
          }}
        />
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

      <CardGrid
        loading={loading}
        isEmpty={universities.length === 0}
        emptyMessage="No universities found"
      >
        {universities.map((u) => (
          <UniversityCard key={u.id} university={u} />
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

function UniversityCard({ university }: { university: University }) {
  const [imageFailed, setImageFailed] = useState(false);
  const title = university.name ?? `University #${university.id}`;
  const href = safeUrl(university.site_url);
  const hasImage = Boolean(university.avatar_url && !imageFailed);

  return (
    <article className="game-card university-card">
      <div className="game-card-content">
        <div className="game-card-heading">
          <Link to={`/universities/${university.id}`} className="game-card-title">
            {title}
          </Link>
        </div>

        <dl className="game-card-meta">
          <div>
            <dt>Site</dt>
            <dd>
              {href ? (
                <a href={href} target="_blank" rel="noreferrer">
                  {formatHost(href)}
                </a>
              ) : (
                (university.site_url ?? "—")
              )}
            </dd>
          </div>
          <div>
            <dt>ID</dt>
            <dd>{university.id}</dd>
          </div>
          <div>
            <dt>Added</dt>
            <dd>{formatDateTime(university.created_at)}</dd>
          </div>
          <div>
            <dt>Updated</dt>
            <dd>{formatDateTime(university.updated_at)}</dd>
          </div>
        </dl>
      </div>

      <Link
        to={`/universities/${university.id}`}
        className="game-card-media"
        tabIndex={-1}
        aria-hidden="true"
      >
        {hasImage ? (
          <img
            src={university.avatar_url ?? ""}
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

function safeUrl(url: string | undefined | null) {
  if (!url) return null;
  try {
    const parsed = new URL(url);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return null;
    }
    return url;
  } catch {
    return null;
  }
}

function formatHost(url: string) {
  try {
    return new URL(url).hostname.replace(/^www\./, "");
  } catch {
    return url;
  }
}

function formatDateTime(value?: string | null) {
  return value ? new Date(value).toLocaleDateString() : "—";
}
