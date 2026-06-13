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
import { useAuth } from "../auth/AuthContext";

const checkBadgeVariant: Record<string, string> = {
  ok: "ok",
  failed: "failed",
  fail: "failed",
  unknown: "unknown",
  queued: "upcoming",
};

export default function ServiceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const serviceId = Number(id);
  const navigate = useNavigate();
  const { isPlayer, isAdmin } = useAuth();

  const [service, setService] = useState<Service | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState<ServiceUpdate>({});
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
    setEditing(true);
  };

  const handleSave = async () => {
    setSaving(true);
    const { data, error: err } = await servicesApi.updateService(
      serviceId,
      editForm,
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

  if (loading) return <div className="loading">Loading...</div>;
  if (!service) return <ErrorDisplay error={error} onRetry={fetchService} />;

  const canEdit = isPlayer;
  const checkVariant = checkBadgeVariant[service.check_status] ?? "unknown";

  return (
    <div className="page detail-page">
      <ErrorDisplay error={error} onRetry={fetchService} />

      {!editing ? (
        <>
          <DetailHero
            kicker={`Service #${service.id}`}
            title={service.name}
            avatarUrl={service.avatar_url}
            avatarText={service.name}
            badges={
              <>
                <CardBadge variant={service.public ? "public" : "private"}>
                  {service.public ? "public" : "private"}
                </CardBadge>
                <CardBadge variant={checkVariant}>
                  check {service.check_status}
                </CardBadge>
              </>
            }
            summary={[
              { label: "Author", value: service.author ?? "—" },
              { label: "Copyright", value: service.copyright ?? "—" },
              {
                label: "Last check",
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
                  Back
                </button>
                {service.writeup_url && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(service.writeup_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Writeup
                  </a>
                )}
                {service.exploits_url && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(service.exploits_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Exploits
                  </a>
                )}
                {canEdit && (
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={startEdit}
                  >
                    Edit
                  </button>
                )}
              </>
            }
          />

          <div className="detail-section">
            <div className="section-head">
              <h3>Service Info</h3>
            </div>
            <InfoGroups>
              <InfoGroup title="Overview">
                <InfoRow label="Author">{service.author ?? "—"}</InfoRow>
                <InfoRow label="Copyright">{service.copyright ?? "—"}</InfoRow>
                <InfoRow label="Visibility">
                  <CardBadge variant={service.public ? "public" : "private"}>
                    {service.public ? "public" : "private"}
                  </CardBadge>
                </InfoRow>
              </InfoGroup>

              <InfoGroup title="Check">
                <InfoRow label="Status">
                  <CardBadge variant={checkVariant}>
                    {service.check_status}
                  </CardBadge>
                </InfoRow>
                <InfoRow label="Checked">
                  {service.checked_at
                    ? formatDateTime(service.checked_at)
                    : "—"}
                </InfoRow>
              </InfoGroup>

              <InfoGroup title="Description">
                <InfoRow label="Public">
                  {service.public_description ?? "—"}
                </InfoRow>
                {isAdmin && service.private_description && (
                  <InfoRow label="Private">
                    {service.private_description}
                  </InfoRow>
                )}
              </InfoGroup>

              <InfoGroup title="Sources">
                <InfoRow label="Service URL">
                  {renderLink(service.service_archive_url)}
                </InfoRow>
                <InfoRow label="Checker URL">
                  {renderLink(service.checker_archive_url)}
                </InfoRow>
                <InfoRow label="Writeup">
                  {renderLink(service.writeup_url)}
                </InfoRow>
                <InfoRow label="Exploits">
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
            <label>Name</label>
            <input
              value={editForm.name ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, name: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Author</label>
            <input
              value={editForm.author ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, author: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Copyright</label>
            <input
              value={editForm.copyright ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, copyright: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Public Description</label>
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
            <label>Private Description</label>
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
            <label>Public</label>
            <input
              type="checkbox"
              checked={editForm.public ?? false}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, public: e.target.checked }))
              }
            />
          </div>
          <div className="form-group">
            <label>Service Archive URL</label>
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
            <label>Checker Archive URL</label>
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
            <label>Writeup URL</label>
            <input
              value={editForm.writeup_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, writeup_url: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Exploits URL</label>
            <input
              value={editForm.exploits_url ?? ""}
              onChange={(e) =>
                setEditForm((f) => ({ ...f, exploits_url: e.target.value }))
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
              onClick={() => setEditing(false)}
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {canEdit && (
        <div className="detail-section">
          <div className="section-head">
            <h3>Actions</h3>
          </div>
          <div className="action-buttons">
            <ActionButton onClick={handleTogglePublic}>
              {service.public ? "Make Private" : "Make Public"}
            </ActionButton>
            <ActionButton onClick={handleCheckChecker}>
              Check Checker
            </ActionButton>
            <ActionButton onClick={handleRedownload}>
              Re-download Archives
            </ActionButton>
            <ActionButton
              onClick={handleDelete}
              variant="danger"
              confirm="Delete this service?"
            >
              Delete
            </ActionButton>
          </div>
        </div>
      )}

      <div className="detail-section">
        <div className="section-head">
          <h3>Archives</h3>
        </div>
        <div className="archive-grid">
          <ArchiveCard
            title="Service archive"
            meta={service.service_archive}
            onDownload={() => void handleDownload("service")}
          />
          <ArchiveCard
            title="Checker archive"
            meta={service.checker_archive}
            onDownload={() => void handleDownload("checker")}
          />
        </div>
        {canEdit && (
          <form onSubmit={(e) => void handleUpload(e)} className="upload-form">
            <div className="form-group">
              <label>Service Archive</label>
              <input
                type="file"
                accept=".zip"
                onChange={(e) =>
                  setServiceArchiveFile(e.target.files?.[0] ?? null)
                }
              />
            </div>
            <div className="form-group">
              <label>Checker Archive</label>
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
              {uploading ? "Uploading..." : "Upload Archives"}
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
  onDownload,
}: {
  title: string;
  meta?: Service["service_archive"];
  onDownload: () => void;
}) {
  const present = Boolean(meta);
  return (
    <div className={`archive-card${present ? "" : " is-empty"}`}>
      <div className="archive-card-head">
        <span className="archive-card-title">{title}</span>
        <CardBadge variant={present ? "ok" : "unknown"}>
          {present ? "present" : "none"}
        </CardBadge>
      </div>
      {present ? (
        <dl className="archive-card-meta">
          <div>
            <dt>Size</dt>
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
        <p className="archive-card-empty">Not uploaded</p>
      )}
      <button className="btn btn-sm" onClick={onDownload} disabled={!present}>
        Download
      </button>
    </div>
  );
}

function formatSize(bytes: number | null | undefined): string {
  if (bytes == null) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
