import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import * as servicesApi from "../api/services";
import type {
  Service,
  ServiceCreate,
  ServiceImportPreview,
} from "../api/services";
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
import { useI18n } from "../i18n/I18nContext";

const checkBadgeVariant: Record<string, string> = {
  ok: "ok",
  failed: "failed",
  fail: "failed",
  unknown: "unknown",
  queued: "upcoming",
};

type ImportSource = "github" | "zip";

type ImportResultResponse = {
  service?: Service;
  warnings?: string[];
};

const importStatusLabel: Record<string, string> = {
  ok: "OK",
  warning: "Warning",
  error: "Error",
};

export default function ServicesPage() {
  const { t } = useI18n();
  usePageTitle(t("Services"));
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
  const [portsInput, setPortsInput] = useState("");
  const [techInput, setTechInput] = useState("");
  const [creating, setCreating] = useState(false);

  const [showImportWizard, setShowImportWizard] = useState(false);
  const [importSource, setImportSource] = useState<ImportSource>("github");
  const [githubUrl, setGithubUrl] = useState("");
  const [githubRef, setGithubRef] = useState("");
  const [githubSubdir, setGithubSubdir] = useState("");
  const [preview, setPreview] = useState<ServiceImportPreview | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const [importing, setImporting] = useState(false);
  const [zipFile, setZipFile] = useState<File | null>(null);

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
    const body: ServiceCreate = { ...createForm };
    const ports = servicesApi.parsePorts(portsInput);
    if (ports.length) body.ports = ports;
    const tech = servicesApi.parseTechStack(techInput);
    if (tech.length) body.tech_stack = tech;
    const { data, error: err } = await servicesApi.createService(body);
    setCreating(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) navigate(`/services/${data.id}`);
  };

  const resetImportPreview = () => {
    setPreview(null);
  };

  const handlePreviewImport = async (e: React.FormEvent) => {
    e.preventDefault();
    setPreviewing(true);
    setError(null);
    setPreview(null);
    try {
      if (importSource === "github") {
        const { data, error: err } =
          await servicesApi.previewServiceGithubImport({
            repo_url: githubUrl,
            ref: githubRef || undefined,
            subdir: githubSubdir || undefined,
          });
        if (err) {
          setError(handleApiError(err));
          return;
        }
        if (data) setPreview(data);
        return;
      }

      if (!zipFile) return;
      const formData = new FormData();
      formData.append("archive", zipFile);
      const response = await servicesApi.previewServiceZipImport(formData);
      const body = await response.json();
      if (!response.ok) {
        setError(handleApiError(body));
        return;
      }
      setPreview(body as ServiceImportPreview);
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setPreviewing(false);
    }
  };

  const handleImport = async () => {
    if (!preview?.valid) return;
    setImporting(true);
    setError(null);
    try {
      if (importSource === "github") {
        const { data, error: err } = await servicesApi.importServiceFromGithub({
          repo_url: githubUrl,
          ref: githubRef || undefined,
          subdir: githubSubdir || undefined,
        });
        if (err) {
          setError(handleApiError(err));
          return;
        }
        if (data) navigate(`/services/${data.service.id}`);
        return;
      }

      if (!zipFile) return;
      const formData = new FormData();
      formData.append("archive", zipFile);
      const response = await servicesApi.importServiceFromZip(formData);
      const result = (await response.json()) as ImportResultResponse;
      if (!response.ok) {
        setError(handleApiError(result));
        return;
      }
      if (result.service) navigate(`/services/${result.service.id}`);
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setImporting(false);
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
            <option value="">{t("All")}</option>
            <option value="true">{t("Public")}</option>
            <option value="false">{t("Private")}</option>
          </select>
          <input
            placeholder={t("Search services...")}
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
              onClick={() => {
                setShowCreate(!showCreate);
                setShowImportWizard(false);
              }}
            >
              {showCreate ? t("Cancel") : t("Create Service")}
            </button>
            <button
              className="btn"
              onClick={() => {
                setShowImportWizard(!showImportWizard);
                setShowCreate(false);
              }}
            >
              {showImportWizard ? t("Close Import") : t("Import Service")}
            </button>
          </div>
        )}
      </div>

      {showCreate && (
        <form onSubmit={(e) => void handleCreate(e)} className="create-form">
          <div className="form-group">
            <label>{t("Name *")}</label>
            <input
              value={createForm.name}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, name: e.target.value }))
              }
              required
            />
          </div>
          <div className="form-group">
            <label>{t("Author")}</label>
            <input
              value={createForm.author ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, author: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Public Description")}</label>
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
            <label>{t("Private Description")}</label>
            <textarea
              value={createForm.private_description ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({
                  ...f,
                  private_description: e.target.value,
                }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Copyright")}</label>
            <input
              value={createForm.copyright ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, copyright: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Avatar URL")}</label>
            <input
              type="url"
              value={createForm.avatar_url ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, avatar_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Writeup URL")}</label>
            <input
              type="url"
              value={createForm.writeup_url ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, writeup_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Exploits URL")}</label>
            <input
              type="url"
              value={createForm.exploits_url ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, exploits_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Service Archive URL")}</label>
            <input
              type="url"
              value={createForm.service_archive_url ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({
                  ...f,
                  service_archive_url: e.target.value,
                }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Checker Archive URL")}</label>
            <input
              type="url"
              value={createForm.checker_archive_url ?? ""}
              onChange={(e) =>
                setCreateForm((f) => ({
                  ...f,
                  checker_archive_url: e.target.value,
                }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Ports")}</label>
            <input
              placeholder={t("e.g. 8080, 9000")}
              value={portsInput}
              onChange={(e) => setPortsInput(e.target.value)}
            />
          </div>
          <div className="form-group">
            <label>{t("Tech stack")}</label>
            <input
              placeholder={t("e.g. Python, PostgreSQL, nginx")}
              value={techInput}
              onChange={(e) => setTechInput(e.target.value)}
            />
          </div>
          <div className="form-group">
            <label>{t("Public")}</label>
            <input
              type="checkbox"
              checked={createForm.public ?? false}
              onChange={(e) =>
                setCreateForm((f) => ({ ...f, public: e.target.checked }))
              }
            />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? t("Creating...") : t("Create")}
          </button>
        </form>
      )}

      {showImportWizard && (
        <form
          onSubmit={(e) => void handlePreviewImport(e)}
          className="import-wizard"
        >
          <div className="import-wizard-top">
            <div className="import-source-tabs">
              <button
                type="button"
                className={`tab ${importSource === "github" ? "active" : ""}`}
                onClick={() => {
                  setImportSource("github");
                  resetImportPreview();
                }}
              >
                {t("GitHub")}
              </button>
              <button
                type="button"
                className={`tab ${importSource === "zip" ? "active" : ""}`}
                onClick={() => {
                  setImportSource("zip");
                  resetImportPreview();
                }}
              >
                {t("ZIP")}
              </button>
            </div>

            <div className="import-steps" aria-label={t("Import steps")}>
              <span
                className={`import-step ${
                  preview ? "is-complete" : "is-active"
                }`}
              >
                {t("Source")}
              </span>
              <span
                className={`import-step ${
                  preview?.valid ? "is-complete" : preview ? "is-active" : ""
                }`}
              >
                {t("Validate")}
              </span>
              <span
                className={`import-step ${preview?.valid ? "is-active" : ""}`}
              >
                {t("Import")}
              </span>
            </div>
          </div>

          {importSource === "github" ? (
            <div className="import-fields">
              <div className="form-group">
                <label>{t("Repo URL *")}</label>
                <input
                  value={githubUrl}
                  onChange={(e) => {
                    setGithubUrl(e.target.value);
                    resetImportPreview();
                  }}
                  required
                  placeholder="https://github.com/SibirCTF/2026-cybersibir-service-name"
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>{t("Ref")}</label>
                  <input
                    value={githubRef}
                    onChange={(e) => {
                      setGithubRef(e.target.value);
                      resetImportPreview();
                    }}
                    placeholder="main"
                  />
                </div>
                <div className="form-group">
                  <label>{t("Subdirectory")}</label>
                  <input
                    value={githubSubdir}
                    onChange={(e) => {
                      setGithubSubdir(e.target.value);
                      resetImportPreview();
                    }}
                  />
                </div>
              </div>
            </div>
          ) : (
            <div className="form-group">
              <label>{t("ZIP Archive *")}</label>
              <input
                type="file"
                accept=".zip"
                onChange={(e) => {
                  setZipFile(e.target.files?.[0] ?? null);
                  resetImportPreview();
                }}
                required
              />
            </div>
          )}

          <div className="form-actions">
            <button
              type="submit"
              className="btn"
              disabled={
                previewing || (importSource === "zip" ? !zipFile : !githubUrl)
              }
            >
              {previewing ? t("Validating...") : t("Validate")}
            </button>
            <button
              type="button"
              className="btn btn-primary"
              disabled={!preview?.valid || importing || previewing}
              onClick={() => void handleImport()}
            >
              {importing ? t("Importing...") : t("Import")}
            </button>
          </div>

          {preview && (
            <div className="import-preview">
              <div className="import-preview-meta">
                <div>
                  <span>{t("Service ID")}</span>
                  <strong>{preview.service_name || "—"}</strong>
                </div>
                <div>
                  <span>{t("Expected repo")}</span>
                  <strong>{preview.expected_repository_name}</strong>
                </div>
                <div>
                  <span>{t("Service dir")}</span>
                  <strong>{preview.service_directory ?? "—"}</strong>
                </div>
                <div>
                  <span>{t("Checker dir")}</span>
                  <strong>{preview.checker_directory ?? "—"}</strong>
                </div>
              </div>

              <div className="import-check-list">
                {preview.requirements.map((item) => (
                  <div
                    key={item.id}
                    className={`import-check import-check-${item.status}`}
                  >
                    <span className="import-check-status">
                      {t(importStatusLabel[item.status] ?? item.status)}
                    </span>
                    <div>
                      <strong>{item.title}</strong>
                      <p>{item.message}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </form>
      )}

      <ErrorDisplay error={error} onRetry={fetchServices} />

      <CardGrid
        loading={loading}
        isEmpty={services.length === 0}
        emptyMessage={t("No services found")}
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
                  {t(s.public ? "public" : "private")}
                </CardBadge>
                <CardBadge
                  variant={checkBadgeVariant[s.check_status] ?? "unknown"}
                >
                  {t(s.check_status)}
                </CardBadge>
              </>
            }
          >
            <CardMeta label={t("Author")}>{s.author ?? "—"}</CardMeta>
            {s.public_description && (
              <CardMeta label={t("About")}>{s.public_description}</CardMeta>
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
