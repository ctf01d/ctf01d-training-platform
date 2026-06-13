import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import * as universitiesApi from "../api/universities";
import type { University, UniversityUpdate } from "../api/universities";
import * as teamsApi from "../api/teams";
import type { Team } from "../api/teams";
import { ErrorDisplay, ActionButton } from "../components/ErrorDisplay";
import {
  DetailHero,
  InfoGroups,
  InfoGroup,
  InfoRow,
  SectionCount,
  renderLink,
  renderLogo,
  formatDateTime,
  safeHref,
} from "../components/DetailInfo";
import { useAuth } from "../auth/AuthContext";

export default function UniversityDetailPage() {
  const { id } = useParams<{ id: string }>();
  const universityId = Number(id);
  const navigate = useNavigate();
  const { isAdmin } = useAuth();

  const [university, setUniversity] = useState<University | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState<UniversityUpdate>({});
  const [saving, setSaving] = useState(false);

  const [teams, setTeams] = useState<Team[]>([]);

  const fetchUniversity = useCallback(async () => {
    setLoading(true);
    const { data, error: err } = await universitiesApi.getUniversity(
      universityId,
    );
    if (err) setError(err);
    else if (data) setUniversity(data);
    setLoading(false);
  }, [universityId]);

  const fetchTeams = useCallback(async () => {
    const { data } = await teamsApi.listTeams({ per_page: 200 });
    if (data)
      setTeams(data.items.filter((t) => t.university_id === universityId));
  }, [universityId]);

  useEffect(() => {
    void fetchUniversity();
    void fetchTeams();
  }, [fetchUniversity, fetchTeams]);

  const startEdit = () => {
    if (!university) return;
    setEditForm({
      name: university.name ?? undefined,
      site_url: university.site_url ?? undefined,
      avatar_url: university.avatar_url ?? undefined,
    });
    setEditing(true);
  };

  const handleSave = async () => {
    setSaving(true);
    const { data, error: err } = await universitiesApi.updateUniversity(
      universityId,
      editForm,
    );
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      setUniversity(data);
      setEditing(false);
    }
  };

  const handleDelete = async () => {
    const { error: err } = await universitiesApi.deleteUniversity(universityId);
    if (err) {
      setError(err);
      return;
    }
    navigate("/universities");
  };

  if (loading) return <div className="loading">Loading...</div>;
  if (!university)
    return <ErrorDisplay error={error} onRetry={fetchUniversity} />;

  const title = university.name ?? `University #${university.id}`;

  return (
    <div className="page detail-page">
      <ErrorDisplay error={error} onRetry={fetchUniversity} />

      {!editing ? (
        <>
          <DetailHero
            kicker={`University #${university.id}`}
            title={title}
            avatarUrl={university.avatar_url}
            avatarText={title}
            summary={[
              { label: "Teams", value: `${teams.length}` },
              {
                label: "Site",
                value: university.site_url
                  ? renderLink(university.site_url)
                  : "—",
              },
            ]}
            actions={
              <>
                <button
                  className="btn btn-sm"
                  onClick={() => navigate("/universities")}
                >
                  Back
                </button>
                {university.site_url && (
                  <a
                    className="btn btn-sm"
                    href={safeHref(university.site_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Site
                  </a>
                )}
                {isAdmin && (
                  <button className="btn btn-sm btn-primary" onClick={startEdit}>
                    Edit
                  </button>
                )}
                {isAdmin && (
                  <ActionButton
                    onClick={handleDelete}
                    variant="danger"
                    confirm="Delete this university?"
                  >
                    Delete
                  </ActionButton>
                )}
              </>
            }
          />

          <div className="detail-section">
            <div className="section-head">
              <h3>University Info</h3>
            </div>
            <InfoGroups>
              <InfoGroup title="Profile">
                <InfoRow label="Name">{university.name ?? "—"}</InfoRow>
                <InfoRow label="Site">{renderLink(university.site_url)}</InfoRow>
                <InfoRow label="Logo">{renderLogo(university.avatar_url)}</InfoRow>
              </InfoGroup>

              <InfoGroup title="Meta">
                <InfoRow label="Added">
                  {formatDateTime(university.created_at)}
                </InfoRow>
                <InfoRow label="Updated">
                  {formatDateTime(university.updated_at)}
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
              onClick={() => setEditing(false)}
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      <div className="detail-section">
        <div className="section-head">
          <h3>
            Teams <SectionCount n={teams.length} />
          </h3>
        </div>
        {teams.length > 0 ? (
          <div className="chip-list">
            {teams.map((t) => (
              <span className="entity-chip" key={t.id}>
                <Link to={`/teams/${t.id}`}>{t.name}</Link>
              </span>
            ))}
          </div>
        ) : (
          <p className="section-empty">No teams from this university.</p>
        )}
      </div>
    </div>
  );
}
