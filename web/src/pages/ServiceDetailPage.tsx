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
  const [saving, setSaving] = useState(false);

  const [serviceArchiveFile, setServiceArchiveFile] = useState<File | null>(
    null,
  );
  const [checkerArchiveFile, setCheckerArchiveFile] = useState<File | null>(
    null,
  );
  const [uploading, setUploading] = useState(false);

  const fetchService = useCallback(async () => {
    setLoading(true);
    const { data, error: err } = await servicesApi.getService(serviceId);
    if (err) setError(err);
    else if (data) setService(data);
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
    setEditing(true);
  };

  const handleSave = async () => {
    setSaving(true);
    const body: ServiceUpdate = {
      ...editForm,
      ports: servicesApi.parsePorts(portsInput),
      tech_stack: servicesApi.parseTechStack(techInput),
    };
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
    const { data, error: err } =
      await servicesApi.toggleServicePublic(serviceId);
    if (err) {
      setError(err);
      return;
    }
    if (data) setService(data);
  };

  const handleCheckChecker = async () => {
    const { data, error: err } =
      await servicesApi.checkServiceChecker(serviceId);
    if (err) {
      setError(err);
      return;
    }
    if (data) setService(data);
  };

  const handleRedownload = async () => {
    const { data, error: err } =
      await servicesApi.redownloadServiceArchives(serviceId);
    if (err) {
      setError(err);
      return;
    }
    if (data) setService(data);
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!serviceArchiveFile && !checkerArchiveFile) return;
    setUploading(true);
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
  const checkVariant = checkBadgeVariant[service.check_status] ?? "unknown";

  return (
    <div className="page detail-page">
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
              { label: t("Author"), value: service.author ?? "—" },
              { label: t("Copyright"), value: service.copyright ?? "—" },
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
                {canEdit && (
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={startEdit}
                  >
                    {t("Edit")}
                  </button>
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

      {canEdit && (
        <div className="detail-section">
          <div className="section-head">
            <h3>{t("Actions")}</h3>
          </div>
          <div className="action-buttons">
            <ActionButton onClick={handleTogglePublic}>
              {service.public ? t("Make Private") : t("Make Public")}
            </ActionButton>
            <ActionButton onClick={handleCheckChecker}>
              {t("Check Checker")}
            </ActionButton>
            <ActionButton onClick={handleRedownload}>
              {t("Re-download Archives")}
            </ActionButton>
            <ActionButton
              onClick={handleDelete}
              variant="danger"
              confirm={t("Delete this service?")}
            >
              {t("Delete")}
            </ActionButton>
          </div>
        </div>
      )}

      <div className="detail-section">
        <div className="section-head">
          <h3>{t("Archives")}</h3>
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
        {canEdit && (
          <form onSubmit={(e) => void handleUpload(e)} className="upload-form">
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
            <button
              type="submit"
              className="btn btn-primary"
              disabled={
                uploading || (!serviceArchiveFile && !checkerArchiveFile)
              }
            >
              {uploading ? t("Uploading...") : t("Upload Archives")}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}

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
      {canDownload && (
        <button className="btn btn-sm" onClick={onDownload} disabled={!present}>
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
