import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import * as servicesApi from "../api/services";
import type { Service, ServiceCreate } from "../api/services";
import {
  CardGrid,
  EntityCard,
  CardBadge,
  CardMeta,
  Pagination,
} from "../components/Card";
import { ErrorDisplay, handleApiError } from "../components/ErrorDisplay";
import { usePageTitle } from "../components/usePageTitle";
import { useAuth } from "../auth/AuthContext";

const checkBadgeVariant: Record<string, string> = {
  ok: "ok",
  failed: "failed",
  fail: "failed",
  unknown: "unknown",
  queued: "upcoming",
};

export default function ServicesPage() {
  usePageTitle("Services");
  const { isPlayer } = useAuth();
  const navigate = useNavigate();
  const [services, setServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 20;
  const [publicFilter, setPublicFilter] = useState<boolean | undefined>(
    undefined,
  );
  const [searchQuery, setSearchQuery] = useState("");

  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState<ServiceCreate>({ name: "" });
  const [creating, setCreating] = useState(false);

  const [showGithubImport, setShowGithubImport] = useState(false);
  const [githubUrl, setGithubUrl] = useState("");
  const [githubRef, setGithubRef] = useState("");
  const [githubSubdir, setGithubSubdir] = useState("");
  const [importing, setImporting] = useState(false);

  const [showZipImport, setShowZipImport] = useState(false);
  const [zipFile, setZipFile] = useState<File | null>(null);
  const [importingZip, setImportingZip] = useState(false);

  const fetchServices = useCallback(async () => {
    setLoading(true);
    setError(null);
    const { data, error: err } = await servicesApi.listServices({
      page,
      per_page: perPage,
      public: publicFilter,
      q: searchQuery || undefined,
    });
    if (err) setError(err);
    else if (data) {
      setServices(data.items);
      setTotal(data.pagination.total);
    }
    setLoading(false);
  }, [page, publicFilter, searchQuery]);

  useEffect(() => {
    void fetchServices();
  }, [fetchServices]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    const { data, error: err } = await servicesApi.createService(createForm);
    setCreating(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) navigate(`/services/${data.id}`);
  };

  const handleGithubImport = async (e: React.FormEvent) => {
    e.preventDefault();
    setImporting(true);
    const { data, error: err } = await servicesApi.importServiceFromGithub({
      repo_url: githubUrl,
      ref: githubRef || undefined,
      subdir: githubSubdir || undefined,
    });
    setImporting(false);
    if (err) {
      setError(handleApiError(err));
      return;
    }
    if (data) navigate(`/services/${data.service.id}`);
  };

  const handleZipImport = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!zipFile) return;
    setImportingZip(true);
    try {
      const formData = new FormData();
      formData.append("archive", zipFile);
      const response = await servicesApi.importServiceFromZip(formData);
      if (!response.ok) {
        const body = await response.json();
        setError(handleApiError(body));
        return;
      }
      const result = await response.json();
      if (result.service) navigate(`/services/${result.service.id}`);
    } catch (e) {
      setError(handleApiError(e));
    } finally {
      setImportingZip(false);
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <div className="filters">
          <select
            value={
              publicFilter === undefined ? "" : publicFilter ? "true" : "false"
            }
            onChange={(e) => {
              setPublicFilter(
                e.target.value === "" ? undefined : e.target.value === "true",
              );
              setPage(1);
            }}
          >
            <option value="">All</option>
            <option value="true">Public</option>
            <option value="false">Private</option>
          </select>
          <input
            placeholder="Search services..."
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        {isPlayer && (
          <div className="action-buttons">
            <button
              className="btn btn-primary"
              onClick={() => setShowCreate(!showCreate)}
            >
              {showCreate ? "Cancel" : "Create Service"}
            </button>
            <button
              className="btn"
              onClick={() => setShowGithubImport(!showGithubImport)}
            >
              {showGithubImport ? "Cancel" : "Import from GitHub"}
            </button>
            <button
              className="btn"
              onClick={() => setShowZipImport(!showZipImport)}
            >
              {showZipImport ? "Cancel" : "Import ZIP"}
            </button>
          </div>
        )}
      </div>

      {showCreate && (
        <form onSubmit={(e) => void handleCreate(e)} className="create-form">
          <div className="form-group">
            <label>Name *</label>
            <input
              value={createForm.name}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, name: e.target.value }))
              }
              required
            />
          </div>
          <div className="form-group">
            <label>Author</label>
            <input
              value={createForm.author ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, author: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Public Description</label>
            <textarea
              value={createForm.public_description ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({
                  ...f,
                  public_description: e.target.value,
                }))
              }
            />
          </div>
          <div className="form-group">
            <label>Public</label>
            <input
              type="checkbox"
              checked={createForm.public ?? false}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, public: e.target.checked }))
              }
            />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? "Creating..." : "Create"}
          </button>
        </form>
      )}

      {showGithubImport && (
        <form
          onSubmit={(e) => void handleGithubImport(e)}
          className="create-form"
        >
          <div className="form-group">
            <label>Repo URL *</label>
            <input
              value={githubUrl}
              onChange={(e) => setGithubUrl(e.target.value)}
              required
              placeholder="https://github.com/org/repo"
            />
          </div>
          <div className="form-group">
            <label>Ref (branch/tag)</label>
            <input
              value={githubRef}
              onChange={(e) => setGithubRef(e.target.value)}
              placeholder="main"
            />
          </div>
          <div className="form-group">
            <label>Subdirectory</label>
            <input
              value={githubSubdir}
              onChange={(e) => setGithubSubdir(e.target.value)}
            />
          </div>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={importing}
          >
            {importing ? "Importing..." : "Import"}
          </button>
        </form>
      )}

      {showZipImport && (
        <form onSubmit={(e) => void handleZipImport(e)} className="create-form">
          <div className="form-group">
            <label>ZIP Archive *</label>
            <input
              type="file"
              accept=".zip"
              onChange={(e) => setZipFile(e.target.files?.[0] ?? null)}
              required
            />
          </div>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={importingZip}
          >
            {importingZip ? "Importing..." : "Import"}
          </button>
        </form>
      )}

      <ErrorDisplay error={error} onRetry={fetchServices} />

      <CardGrid
        loading={loading}
        isEmpty={services.length === 0}
        emptyMessage="No services found"
      >
        {services.map((s) => (
          <EntityCard
            key={s.id}
            to={`/services/${s.id}`}
            avatarUrl={s.avatar_url}
            avatarText={s.name}
            title={s.name}
            badges={
              <>
                <CardBadge variant={s.public ? "public" : "private"}>
                  {s.public ? "public" : "private"}
                </CardBadge>
                <CardBadge
                  variant={checkBadgeVariant[s.check_status] ?? "unknown"}
                >
                  {s.check_status}
                </CardBadge>
              </>
            }
          >
            <CardMeta label="Author">{s.author ?? "—"}</CardMeta>
            {s.public_description && (
              <CardMeta label="About">{s.public_description}</CardMeta>
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
