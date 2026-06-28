import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import * as servicesApi from "../api/services";
import type { Service, ServiceUpdate } from "../api/services";
import {
  ErrorDisplay,
  ActionButton,
  handleApiError,
} from "../components/ErrorDisplay";
import { CardBadge } from "../components/Card";
import {
  DetailHero,
  InfoGroups,
  InfoGroup,
  InfoRow,
  renderLink,
  formatDateTime,
  safeHref,
} from "../components/DetailInfo";
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

export default function ServiceDetailPage() {
  const { t } = useI18n();
  const { id } = useParams<{ id: string }>();
  const serviceId = Number(id);
  const navigate = useNavigate();
  const { user, isPlayer, isAdmin } = useAuth();

  const [service, setService] = useState<Service | null>(null);
  usePageTitle(service?.name);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState<ServiceUpdate>({});
  const [portsInput, setPortsInput] = useState("");
  const [techInput, setTechInput] = useState("");
  const [gitRepoUrl, setGitRepoUrl] = useState("");
  const [gitRef, setGitRef] = useState("");
  const [gitSubdir, setGitSubdir] = useState("");
  const [saving, setSaving] = useState(false);
  const [togglingPublic, setTogglingPublic] = useState(false);
  const [checkingChecker, setCheckingChecker] = useState(false);
  const [redownloading, setRedownloading] = useState(false);
  const [syncingGit, setSyncingGit] = useState(false);

  const [serviceArchiveFile, setServiceArchiveFile] = useState<File | null>(
    null,
  );
  const [checkerArchiveFile, setCheckerArchiveFile] = useState<File | null>(
    null,
  );
  const [showUploadForm, setShowUploadForm] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [syncResult, setSyncResult] = useState<SyncResultState | null>(null);

  const fetchService = useCallback(async () => {
    setLoading(true);
    const { data, error: err } = await servicesApi.getService(serviceId);
    if (err) setError(err);
    else if (data) {
      setService(data);
      setError(null);
    }
    setLoading(false);
  }, [serviceId]);

  useEffect(() => {
    void fetchService();
  }, [fetchService]);

  const startEdit = () => {
    if (!service) return;
    setEditForm({
      name: service.name,
      public_description: service.public_description ?? undefined,
      private_description: service.private_description ?? undefined,
      author: service.author ?? undefined,
      copyright: service.copyright ?? undefined,
      avatar_url: service.avatar_url ?? undefined,
      public: service.public,
      service_archive_url: service.service_archive_url ?? undefined,
      checker_archive_url: service.checker_archive_url ?? undefined,
      writeup_url: service.writeup_url ?? undefined,
      exploits_url: service.exploits_url ?? undefined,
    });
    setPortsInput((service.ports ?? []).join(", "));
    setTechInput((service.tech_stack ?? []).join(", "));
    setGitRepoUrl(service.source?.repo_url ?? "");
    setGitRef(service.source?.ref ?? "");
    setGitSubdir(service.source?.subdir ?? "");
    setShowUploadForm(false);
    setServiceArchiveFile(null);
    setCheckerArchiveFile(null);
    setEditing(true);
  };

  const handleSave = async () => {
    setSaving(true);
    const body: ServiceUpdate = {
      ...editForm,
      ports: servicesApi.parsePorts(portsInput),
      tech_stack: servicesApi.parseTechStack(techInput),
    };
    if (isAdmin) {
      body.git_source = {
        repo_url: gitRepoUrl || undefined,
        ref: gitRef || undefined,
        subdir: gitSubdir || undefined,
      };
    }
    const { data, error: err } = await servicesApi.updateService(
      serviceId,
      body,
    );
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      setService(data);
      setEditing(false);
    }
  };

  const handleTogglePublic = async () => {
    setTogglingPublic(true);
    setError(null);
    try {
      const { data, error: err } =
        await servicesApi.toggleServicePublic(serviceId);
      if (err) {
        setError(err);
        return;
      }
      if (data) setService(data);
    } finally {
      setTogglingPublic(false);
    }
  };

  const handleCheckChecker = async () => {
    setCheckingChecker(true);
    setError(null);
    try {
      const { data, error: err } =
        await servicesApi.checkServiceChecker(serviceId);
      if (err) {
        setError(err);
        return;
      }
      if (data) setService(data);
    } finally {
      setCheckingChecker(false);
    }
  };

  const handleRedownload = async () => {
    setRedownloading(true);
    setError(null);
    try {
      const { data, error: err } =
        await servicesApi.redownloadServiceArchives(serviceId);
      if (err) {
        setError(err);
        return;
      }
      if (data) setService(data);
    } finally {
      setRedownloading(false);
    }
  };

  const handleSyncFromGit = async () => {
    setSyncingGit(true);
    setError(null);
    setSyncResult(null);
    try {
      const { data, error: err } = await servicesApi.syncServiceFromGit(
        serviceId,
      );
      if (err) {
        const normalized = handleApiError(err);
        await fetchService();
        setSyncResult(buildSyncFailureResult(normalized));
        return;
      }
      if (data) {
        setService(data);
        setSyncResult({
          status: "success",
          serviceName: data.name,
          lastCommit: data.source?.last_commit ?? null,
          syncedAt: data.source?.synced_at ?? null,
        });
      }
    } finally {
      setSyncingGit(false);
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!serviceArchiveFile && !checkerArchiveFile) return;
    setUploading(true);
    setError(null);
    try {
      const formData = new FormData();
      if (serviceArchiveFile)
        formData.append("service_archive", serviceArchiveFile);
      if (checkerArchiveFile)
        formData.append("checker_archive", checkerArchiveFile);
      const token = localStorage.getItem("auth_token");
      const response = await fetch(
        `/api/v1/services/${serviceId}/upload-archives`,
        {
          method: "POST",
          headers: token ? { Authorization: `Bearer ${token}` } : {},
          body: formData,
        },
      );
      if (!response.ok) {
        const body = await response.json();
        setError(handleApiError(body));
        return;
      }
      const data = await response.json();
      if (data) setService(data);
      setServiceArchiveFile(null);
      setCheckerArchiveFile(null);
      setShowUploadForm(false);
    } catch (e) {
      setError(handleApiError(e));
    } finally {
      setUploading(false);
    }
  };

  const handleDownload = async (kind: "service" | "checker") => {
    const { data, error: err } = await servicesApi.downloadServiceArchive(
      serviceId,
      kind,
    );
    if (err) {
      setError(handleApiError(err));
      return;
    }
    if (data) {
      const blob = data as unknown as Blob;
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${kind}-archive-${serviceId}.zip`;
      a.click();
      URL.revokeObjectURL(url);
    }
  };

  const handleDelete = async () => {
    const { error: err } = await servicesApi.deleteService(serviceId);
    if (err) {
      setError(err);
      return;
    }
    navigate("/services");
  };

  if (loading) return <div className="loading">{t("Loading...")}</div>;
  if (!service) return <ErrorDisplay error={error} onRetry={fetchService} />;

  const canEdit = isPlayer;
  const canEditGitSource = isAdmin;
  const canSyncFromGit = isAdmin && service.source?.kind === "git";
  const canRedownloadArchives = Boolean(
    service.service_archive_url || service.checker_archive_url,
  );
  const checkVariant = checkBadgeVariant[service.check_status] ?? "unknown";
  const actionBusy =
    saving ||
    togglingPublic ||
    checkingChecker ||
    redownloading ||
    syncingGit ||
    uploading;
  const serviceArchiveSummary = service.service_archive
    ? `${t("present")} · ${formatSize(service.service_archive.size)}`
    : t("none");
  const checkerArchiveSummary = service.checker_archive
    ? `${t("present")} · ${formatSize(service.checker_archive.size)}`
    : t("none");

  return (
    <div className="page detail-page service-detail-page">
      <ErrorDisplay error={error} onRetry={fetchService} />

      {!editing ? (
        <>
          <DetailHero
            kicker={`${t("Service")} #${service.id}`}
            title={service.name}
            avatarUrl={service.avatar_url}
            avatarText={service.name}
            badges={
              <>
                <CardBadge variant={service.public ? "public" : "private"}>
                  {service.public ? t("public") : t("private")}
                </CardBadge>
                <CardBadge variant={checkVariant}>
                  {t("Check")} {t(service.check_status)}
                </CardBadge>
              </>
            }
            summary={[
              {
                label: t("Service archive"),
                value: serviceArchiveSummary,
              },
              {
                label: t("Checker archive"),
                value: checkerArchiveSummary,
              },
              {
                label: t("Last check"),
                value: service.checked_at
                  ? formatDateTime(service.checked_at)
                  : "—",
              },
            ]}
            actions={
              <>
                <button
                  className="btn btn-sm"
                  onClick={() => navigate("/services")}
                >
                  {t("Back")}
                </button>
                {canEdit && (
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={startEdit}
                    disabled={actionBusy}
                  >
                    {t("Edit")}
                  </button>
                )}
                {service.writeup_url && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(service.writeup_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {t("Writeup")}
                  </a>
                )}
                {service.exploits_url && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(service.exploits_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {t("Exploits")}
                  </a>
                )}
              </>
            }
          />

          <div className="detail-section">
            <div className="section-head">
              <h3>{t("Service Info")}</h3>
            </div>
            <InfoGroups>
              <InfoGroup title={t("Overview")}>
                <InfoRow label={t("Author")}>{service.author ?? "—"}</InfoRow>
                <InfoRow label={t("Copyright")}>
                  {service.copyright ?? "—"}
                </InfoRow>
                <InfoRow label={t("Ports")}>
                  {service.ports && service.ports.length
                    ? service.ports.join(", ")
                    : "—"}
                </InfoRow>
                <InfoRow label={t("Tech stack")}>
                  {service.tech_stack && service.tech_stack.length
                    ? service.tech_stack.join(", ")
                    : "—"}
                </InfoRow>
                <InfoRow label={t("Visibility")}>
                  <CardBadge variant={service.public ? "public" : "private"}>
                    {service.public ? t("public") : t("private")}
                  </CardBadge>
                </InfoRow>
              </InfoGroup>

              <InfoGroup title={t("Check")}>
                <InfoRow label={t("Status")}>
                  <CardBadge variant={checkVariant}>
                    {t(service.check_status)}
                  </CardBadge>
                </InfoRow>
                <InfoRow label={t("Checked")}>
                  {service.checked_at
                    ? formatDateTime(service.checked_at)
                    : "—"}
                </InfoRow>
              </InfoGroup>

              <InfoGroup title={t("Description")}>
                <InfoRow label={t("Public")}>
                  {service.public_description ?? "—"}
                </InfoRow>
                {isAdmin && service.private_description && (
                  <InfoRow label={t("Private")}>
                    {service.private_description}
                  </InfoRow>
                )}
              </InfoGroup>

              <InfoGroup title={t("Sources")}>
                <InfoRow label={t("Service URL")}>
                  {renderLink(service.service_archive_url)}
                </InfoRow>
                <InfoRow label={t("Checker URL")}>
                  {renderLink(service.checker_archive_url)}
                </InfoRow>
                <InfoRow label={t("Writeup")}>
                  {renderLink(service.writeup_url)}
                </InfoRow>
                <InfoRow label={t("Exploits")}>
                  {renderLink(service.exploits_url)}
                </InfoRow>
              </InfoGroup>

              {service.source && (
                <InfoGroup title={t("Git Source")}>
                  <InfoRow label={t("Mode")}>
                    <CardBadge
                      variant={
                        service.source.kind === "git"
                          ? "upcoming"
                          : service.source.kind === "zip"
                            ? "unknown"
                            : "unknown"
                      }
                    >
                      {service.source.kind}
                    </CardBadge>
                  </InfoRow>
                  <InfoRow label={t("Repo")}>
                    {renderRepoSource(service.source.repo_url)}
                  </InfoRow>
                  <InfoRow label={t("Ref")}>
                    {service.source.ref ?? "—"}
                  </InfoRow>
                  <InfoRow label={t("Subdirectory")}>
                    {service.source.subdir ?? "—"}
                  </InfoRow>
                  <InfoRow label={t("Last commit")}>
                    {service.source.last_commit ? (
                      <code title={service.source.last_commit}>
                        {service.source.last_commit.slice(0, 12)}
                      </code>
                    ) : (
                      "—"
                    )}
                  </InfoRow>
                  <InfoRow label={t("Sync")}>
                    <CardBadge
                      variant={
                        service.source.sync_status === "ok"
                          ? "ok"
                          : service.source.sync_status === "failed"
                            ? "failed"
                            : "unknown"
                      }
                    >
                      {t(service.source.sync_status)}
                    </CardBadge>
                  </InfoRow>
                  <InfoRow label={t("Synced at")}>
                    {service.source.synced_at
                      ? formatDateTime(service.source.synced_at)
                      : "—"}
                  </InfoRow>
                  {service.source.sync_error && (
                    <InfoRow label={t("Last error")}>
                      {service.source.sync_error}
                    </InfoRow>
                  )}
                </InfoGroup>
              )}
            </InfoGroups>
          </div>
        </>
      ) : (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void handleSave();
          }}
          className="edit-form"
        >
          <div className="form-group">
            <label>{t("Name")}</label>
            <input
              value={editForm.name ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, name: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Author")}</label>
            <input
              value={editForm.author ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, author: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Copyright")}</label>
            <input
              value={editForm.copyright ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, copyright: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Public Description")}</label>
            <textarea
              value={editForm.public_description ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({
                  ...f,
                  public_description: e.target.value,
                }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Private Description")}</label>
            <textarea
              value={editForm.private_description ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({
                  ...f,
                  private_description: e.target.value,
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
              checked={editForm.public ?? false}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, public: e.target.checked }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Service Archive URL")}</label>
            <input
              value={editForm.service_archive_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({
                  ...f,
                  service_archive_url: e.target.value,
                }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Checker Archive URL")}</label>
            <input
              value={editForm.checker_archive_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({
                  ...f,
                  checker_archive_url: e.target.value,
                }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Writeup URL")}</label>
            <input
              value={editForm.writeup_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, writeup_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>{t("Exploits URL")}</label>
            <input
              value={editForm.exploits_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, exploits_url: e.target.value }))
              }
            />
          </div>
          {canEditGitSource && (
            <>
              <div className="form-group">
                <label>{t("Git Repo URL")}</label>
                <input
                  value={gitRepoUrl}
                  onChange={(e) => setGitRepoUrl(e.target.value)}
                  placeholder="git@github.com:team/service.git"
                />
              </div>
              <div className="form-group">
                <label>{t("Git Ref")}</label>
                <input
                  value={gitRef}
                  onChange={(e) => setGitRef(e.target.value)}
                />
              </div>
              <div className="form-group">
                <label>{t("Git Subdirectory")}</label>
                <input
                  value={gitSubdir}
                  onChange={(e) => setGitSubdir(e.target.value)}
                />
              </div>
            </>
          )}
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {saving ? t("Saving...") : t("Save")}
            </button>
            <button
              type="button"
              className="btn"
              onClick={() => setEditing(false)}
            >
              {t("Cancel")}
            </button>
          </div>
        </form>
      )}

      <div className="detail-section">
        <div className="section-head">
          <h3>{t("Archives")}</h3>
          {canEdit && (
            <button
              type="button"
              className="btn btn-sm"
              onClick={() => {
                if (showUploadForm) {
                  setShowUploadForm(false);
                  setServiceArchiveFile(null);
                  setCheckerArchiveFile(null);
                  return;
                }
                setShowUploadForm(true);
              }}
              disabled={uploading || editing}
            >
              {showUploadForm ? t("Cancel") : t("Upload Archives")}
            </button>
          )}
        </div>
        <div className="archive-grid">
          <ArchiveCard
            title={t("Service archive")}
            meta={service.service_archive}
            canDownload={Boolean(user)}
            onDownload={() => void handleDownload("service")}
          />
          <ArchiveCard
            title={t("Checker archive")}
            meta={service.checker_archive}
            canDownload={Boolean(user)}
            onDownload={() => void handleDownload("checker")}
          />
        </div>
        {canEdit && showUploadForm && (
          <form
            onSubmit={(e) => void handleUpload(e)}
            className="upload-form service-upload-form"
          >
            <div className="form-group">
              <label>{t("Service Archive")}</label>
              <input
                type="file"
                accept=".zip"
                onChange={(e) =>
                  setServiceArchiveFile(e.target.files?.[0] ?? null)
                }
              />
            </div>
            <div className="form-group">
              <label>{t("Checker Archive")}</label>
              <input
                type="file"
                accept=".zip"
                onChange={(e) =>
                  setCheckerArchiveFile(e.target.files?.[0] ?? null)
                }
              />
            </div>
            <div className="form-actions">
              <button
                type="submit"
                className="btn btn-primary"
                disabled={uploading || (!serviceArchiveFile && !checkerArchiveFile)}
              >
                {uploading ? t("Uploading...") : t("Upload Archives")}
              </button>
              <button
                type="button"
                className="btn"
                onClick={() => {
                  setShowUploadForm(false);
                  setServiceArchiveFile(null);
                  setCheckerArchiveFile(null);
                }}
                disabled={uploading}
              >
                {t("Cancel")}
              </button>
            </div>
          </form>
        )}
      </div>

      {canEdit && (
        <div className="detail-section">
          <div className="section-head">
            <h3>{t("Service Management")}</h3>
          </div>
          <div className="action-buttons">
            <ActionButton
              onClick={handleTogglePublic}
              disabled={actionBusy || editing}
            >
              {togglingPublic
                ? t("Updating...")
                : service.public
                  ? t("Make Private")
                  : t("Make Public")}
            </ActionButton>
            <ActionButton
              onClick={handleCheckChecker}
              disabled={actionBusy || editing}
            >
              {checkingChecker ? t("Checking...") : t("Check Checker")}
            </ActionButton>
            {canRedownloadArchives && (
              <ActionButton
                onClick={handleRedownload}
                disabled={actionBusy || editing}
              >
                {redownloading
                  ? t("Re-downloading...")
                  : t("Re-download Archives")}
              </ActionButton>
            )}
            {canSyncFromGit && (
              <ActionButton
                onClick={handleSyncFromGit}
                disabled={actionBusy || editing}
              >
                {syncingGit ? t("Synchronizing...") : t("Synchronize")}
              </ActionButton>
            )}
          </div>
        </div>
      )}

      {canEdit && (
        <div className="detail-section">
          <div className="section-head">
            <h3>{t("Danger Zone")}</h3>
          </div>
          <ActionButton
            onClick={handleDelete}
            variant="danger"
            disabled={actionBusy || editing}
            confirm={t("Delete this service?")}
          >
            {t("Delete")}
          </ActionButton>
        </div>
      )}

      {syncResult && (
        <SyncResultModal
          result={syncResult}
          onClose={() => setSyncResult(null)}
        />
      )}
    </div>
  );
}

type SyncResultState =
  | {
      status: "success";
      serviceName: string;
      lastCommit: string | null;
      syncedAt: string | null;
    }
  | {
      status: "warning" | "error";
      items: SyncIssue[];
    };

type SyncIssueSeverity = "warning" | "error";

type SyncIssue = {
  severity: SyncIssueSeverity;
  title: string;
  description: string;
  params?: Record<string, string>;
  subject?: string;
};

function ArchiveCard({
  title,
  meta,
  canDownload,
  onDownload,
}: {
  title: string;
  meta?: Service["service_archive"];
  canDownload: boolean;
  onDownload: () => void;
}) {
  const { t } = useI18n();
  const present = Boolean(meta);
  return (
    <div className={`archive-card${present ? "" : " is-empty"}`}>
      <div className="archive-card-head">
        <span className="archive-card-title">{title}</span>
        <CardBadge variant={present ? "ok" : "unknown"}>
          {present ? t("present") : t("none")}
        </CardBadge>
      </div>
      {present ? (
        <dl className="archive-card-meta">
          <div>
            <dt>{t("Size")}</dt>
            <dd>{formatSize(meta?.size)}</dd>
          </div>
          {meta?.sha256 && (
            <div>
              <dt>SHA-256</dt>
              <dd>
                <code title={meta.sha256}>{meta.sha256.slice(0, 16)}…</code>
              </dd>
            </div>
          )}
        </dl>
      ) : (
        <p className="archive-card-empty">{t("Not uploaded")}</p>
      )}
      {canDownload && present && (
        <button className="btn btn-sm" onClick={onDownload}>
          {t("Download")}
        </button>
      )}
    </div>
  );
}

function formatSize(bytes: number | null | undefined): string {
  if (bytes == null) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function renderRepoSource(repoUrl?: string | null) {
  if (!repoUrl) return "—";
  if (repoUrl.startsWith("http://") || repoUrl.startsWith("https://")) {
    return renderLink(repoUrl);
  }
  return <code>{repoUrl}</code>;
}

function buildSyncFailureResult(error: {
  message?: string;
  details?: Record<string, unknown> | null;
}): Extract<SyncResultState, { status: "warning" | "error" }> {
  const items = extractSyncIssues(error);
  const hasError = items.some((item) => item.severity === "error");
  return {
    status: hasError ? "error" : "warning",
    items,
  };
}

function extractSyncIssues(error: {
  message?: string;
  details?: Record<string, unknown> | null;
}): SyncIssue[] {
  const items: SyncIssue[] = [];

  if (error.details && typeof error.details === "object") {
    for (const [field, value] of Object.entries(error.details)) {
      const text = String(value ?? "").trim();
      if (!text) continue;
      for (const part of text.split(";")) {
        const trimmed = part.trim();
        if (!trimmed) continue;
        items.push(normalizeSyncIssue(field, trimmed));
      }
    }
  }

  if (items.length === 0 && error.message) {
    items.push(normalizeSyncIssue("", error.message));
  }

  const seen = new Set<string>();
  return items.filter((item) => {
    const key = JSON.stringify(item);
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function normalizeSyncIssue(field: string, message: string): SyncIssue {
  const trimmed = message.trim();

  const missingDirectory = /^(.*?) directory is required$/i.exec(trimmed);
  if (missingDirectory) {
    return {
      severity: "warning",
      title: "Missing directory",
      description: "This directory must exist in the repository root.",
      subject: missingDirectory[1],
    };
  }

  const missingFile = /^(.*?) must contain (.*)$/i.exec(trimmed);
  if (missingFile) {
    return {
      severity: "warning",
      title: "Missing required file",
      description: "Add one of the following files: @{files}",
      params: { files: missingFile[2] },
      subject: missingFile[1],
    };
  }

  if (trimmed === "git source is not configured") {
    return {
      severity: "error",
      title: "Git source is not configured",
      description:
        "Configure repository URL, ref, and optional subdirectory before synchronizing.",
    };
  }

  if (looksLikeRepositoryAccessIssue(trimmed)) {
    return {
      severity: "error",
      title: "Repository access failed",
      description: "Could not access repository: @{message}",
      params: { message: trimmed },
    };
  }

  if (looksLikeValidationIssue(trimmed)) {
    return {
      severity: "warning",
      title: "Repository layout issue",
      description: "Synchronization issue: @{message}",
      params: { message: trimmed },
      subject: field === "repo_url" ? undefined : field,
    };
  }

  return {
    severity: "error",
    title: "Synchronization stopped",
    description: "Synchronization issue: @{message}",
    params: { message: trimmed },
    subject: field === "repo_url" ? undefined : field,
  };
}

function looksLikeValidationIssue(message: string): boolean {
  const normalized = message.toLowerCase();
  return (
    normalized.includes("directory is required") ||
    normalized.includes("must contain") ||
    normalized.includes("invalid service name") ||
    normalized.includes("could not determine service name") ||
    normalized.includes("manifest")
  );
}

function looksLikeRepositoryAccessIssue(message: string): boolean {
  const normalized = message.toLowerCase();
  return (
    normalized.includes("authentication") ||
    normalized.includes("permission denied") ||
    normalized.includes("repository not found") ||
    normalized.includes("timed out") ||
    normalized.includes("timeout") ||
    normalized.includes("unable to access") ||
    normalized.includes("could not read") ||
    normalized.includes("dial tcp") ||
    normalized.includes("connection refused")
  );
}

function SyncResultModal({
  result,
  onClose,
}: {
  result: SyncResultState;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const warnings =
    result.status === "success"
      ? []
      : result.items.filter((item) => item.severity === "warning");
  const errors =
    result.status === "success"
      ? []
      : result.items.filter((item) => item.severity === "error");
  const variant =
    result.status === "success"
      ? "success"
      : result.status === "warning"
        ? "warning"
        : "error";

  return (
    <div className="sync-result-modal-backdrop" onClick={onClose}>
      <div
        className={`sync-result-modal sync-result-modal--${variant}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="sync-result-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="sync-result-modal-head">
          <div>
            <p className="sync-result-modal-kicker">
              {t("Synchronization result")}
            </p>
            <h3 id="sync-result-title">
              {result.status === "success"
                ? t("Synchronization complete")
                : result.status === "warning"
                  ? t("Synchronization needs attention")
                  : t("Synchronization failed")}
            </h3>
          </div>
          <CardBadge
            variant={
              result.status === "success"
                ? "ok"
                : result.status === "warning"
                  ? "warning"
                  : "error"
            }
          >
            {result.status === "success"
              ? t("OK")
              : result.status === "warning"
                ? t("Warning")
                : t("Error")}
          </CardBadge>
        </div>

        {result.status === "success" ? (
          <>
            <div className="sync-result-summary sync-result-summary--success">
              <p className="sync-result-modal-text">
                {t("Service synchronized successfully.")}
              </p>
            </div>
            <dl className="sync-result-modal-meta">
              <div>
                <dt>{t("Service")}</dt>
                <dd>{result.serviceName}</dd>
              </div>
              <div>
                <dt>{t("Last commit")}</dt>
                <dd>
                  {result.lastCommit ? (
                    <code title={result.lastCommit}>
                      {result.lastCommit.slice(0, 12)}
                    </code>
                  ) : (
                    "—"
                  )}
                </dd>
              </div>
              <div>
                <dt>{t("Synced at")}</dt>
                <dd>
                  {result.syncedAt ? formatDateTime(result.syncedAt) : "—"}
                </dd>
              </div>
            </dl>
          </>
        ) : (
          <>
            <div
              className={`sync-result-summary sync-result-summary--${result.status}`}
            >
              <p className="sync-result-modal-text">
                {result.status === "warning"
                  ? t(
                      "Repository structure needs attention before synchronization can continue.",
                    )
                  : t(
                      "A technical error interrupted synchronization. Review the problems below.",
                    )}
              </p>
            </div>

            <div className="sync-result-groups">
              {warnings.length > 0 && (
                <section className="sync-result-group">
                  <div className="sync-result-group-head">
                    <h4>{t("Warnings")}</h4>
                    <CardBadge variant="warning">{warnings.length}</CardBadge>
                  </div>
                  <div className="sync-result-issues">
                    {warnings.map((item, index) => (
                      <article
                        key={`${item.title}-${item.subject ?? ""}-${index}`}
                        className="sync-result-issue sync-result-issue--warning"
                      >
                        <CardBadge variant="warning">{t("Warn")}</CardBadge>
                        <div>
                          <strong>{t(item.title)}</strong>
                          {item.subject && (
                            <code className="sync-result-issue-subject">
                              {item.subject}
                            </code>
                          )}
                          <p>{t(item.description, item.params)}</p>
                        </div>
                      </article>
                    ))}
                  </div>
                </section>
              )}

              {errors.length > 0 && (
                <section className="sync-result-group">
                  <div className="sync-result-group-head">
                    <h4>{t("Errors")}</h4>
                    <CardBadge variant="error">{errors.length}</CardBadge>
                  </div>
                  <div className="sync-result-issues">
                    {errors.map((item, index) => (
                      <article
                        key={`${item.title}-${item.subject ?? ""}-${index}`}
                        className="sync-result-issue sync-result-issue--error"
                      >
                        <CardBadge variant="error">{t("Error")}</CardBadge>
                        <div>
                          <strong>{t(item.title)}</strong>
                          {item.subject && (
                            <code className="sync-result-issue-subject">
                              {item.subject}
                            </code>
                          )}
                          <p>{t(item.description, item.params)}</p>
                        </div>
                      </article>
                    ))}
                  </div>
                </section>
              )}
            </div>
          </>
        )}

        <div className="form-actions">
          <button type="button" className="btn btn-primary" onClick={onClose}>
            {t("Close")}
          </button>
        </div>
      </div>
    </div>
  );
}
