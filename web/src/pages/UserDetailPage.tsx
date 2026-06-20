import { useState, useEffect, useCallback, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import * as usersApi from "../api/users";
import type { User, UserSession, UserRole } from "../api/users";
import {
  ErrorDisplay,
  ActionButton,
  handleApiError,
} from "../components/ErrorDisplay";
import { usePageTitle } from "../components/usePageTitle";
import {
  DetailHero,
  InfoGroups,
  InfoGroup,
  InfoRow,
  SectionCount,
  formatDateTime,
  formatRelativeTime,
} from "../components/DetailInfo";
import { useAuth } from "../auth/AuthContext";

type ProfileForm = {
  display_name: string;
  bio: string;
  telegram: string;
  github: string;
  email: string;
};

function emptyProfileForm(): ProfileForm {
  return {
    display_name: "",
    bio: "",
    telegram: "",
    github: "",
    email: "",
  };
}

export default function UserDetailPage() {
  const { id } = useParams<{ id: string }>();
  const userId = Number(id);
  const navigate = useNavigate();
  const { user: currentUser } = useAuth();

  const [user, setUser] = useState<User | null>(null);
  usePageTitle(user?.display_name ?? undefined);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [form, setForm] = useState<ProfileForm>(emptyProfileForm());
  const [saving, setSaving] = useState(false);

  const [passwordForm, setPasswordForm] = useState({ password: "", confirm: "" });
  const [changingPassword, setChangingPassword] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);

  const [sessions, setSessions] = useState<UserSession[]>([]);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleting, setDeleting] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);

  const applyUser = useCallback((u: User) => {
    setUser(u);
    setForm({
      display_name: u.display_name ?? "",
      bio: u.bio ?? "",
      telegram: u.telegram ?? "",
      github: u.github ?? "",
      email: u.email ?? "",
    });
  }, []);

  const fetchUser = useCallback(async () => {
    setLoading(true);
    const { data, error: err } = await usersApi.getUser(userId);
    if (err) setError(err);
    else if (data) applyUser(data);
    setLoading(false);
  }, [userId, applyUser]);

  const fetchSessions = useCallback(async () => {
    const { data } = await usersApi.listUserSessions(userId);
    if (data) setSessions(data.items);
  }, [userId]);

  useEffect(() => {
    void fetchUser();
    void fetchSessions();
  }, [fetchUser, fetchSessions]);

  const handleSaveProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(null);
    setSuccess(null);
    const body: usersApi.UserProfileUpdate = {
      display_name: form.display_name,
      bio: form.bio || null,
      telegram: form.telegram || null,
      github: form.github || null,
      email: form.email || null,
    };
    const { data, error: err } = await usersApi.updateUserProfileAdmin(
      userId,
      body,
    );
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      applyUser(data);
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setSuccess(null);
    if (passwordForm.password !== passwordForm.confirm) {
      setError({ message: "Passwords do not match." });
      return;
    }
    setChangingPassword(true);
    setError(null);
    // Dedicated endpoint: only the password changes, no other profile fields are
    // touched, so the open profile form is left untouched.
    const { error: err } = await usersApi.changeUserPassword(
      userId,
      passwordForm.password,
    );
    setChangingPassword(false);
    if (err) {
      setError(err);
      return;
    }
    setPasswordForm({ password: "", confirm: "" });
    setSuccess("Password updated successfully.");
  };

  const handleRoleChange = async (role: UserRole) => {
    setError(null);
    const { data, error: err } = await usersApi.updateUserRole(userId, role);
    if (err) {
      setError(err);
      return;
    }
    if (data) applyUser(data);
  };

  const handleToggleBlock = async () => {
    if (!user) return;
    setError(null);
    const { data, error: err } = await usersApi.setUserBlocked(
      userId,
      !user.is_blocked,
    );
    if (err) {
      setError(err);
      return;
    }
    if (data) applyUser(data);
    await fetchSessions();
  };

  const handleAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploadingAvatar(true);
    setError(null);
    try {
      const response = await usersApi.uploadUserAvatar(userId, file);
      if (!response.ok) {
        const body = await response.json();
        setError(handleApiError(body));
        return;
      }
      const data = (await response.json()) as User;
      applyUser(data);
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setUploadingAvatar(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  const handleRevokeSession = async (sessionId: number) => {
    const { error: err } = await usersApi.revokeUserSession(userId, sessionId);
    if (err) {
      setError(err);
      return;
    }
    await fetchSessions();
  };

  const handleDelete = async (e: React.FormEvent) => {
    e.preventDefault();
    setDeleting(true);
    setError(null);
    const { error: err } = await usersApi.deleteUser(userId, deletePassword);
    setDeleting(false);
    if (err) {
      setError(err);
      return;
    }
    navigate("/users");
  };

  if (loading) return <div className="loading">Loading...</div>;
  if (!user) return <ErrorDisplay error={error} onRetry={fetchUser} />;

  const title = user.display_name || user.user_name;

  return (
    <div className="page detail-page">
      <ErrorDisplay error={error} onRetry={fetchUser} />
      {success && <div className="success-message">{success}</div>}

      <DetailHero
        kicker={`User #${user.id}`}
        title={title}
        avatarUrl={user.avatar_url}
        avatarText={title}
        summary={[
          { label: "Username", value: `@${user.user_name}` },
          { label: "Role", value: user.role },
          { label: "Rating", value: `${user.rating}` },
          { label: "Status", value: user.is_blocked ? "Blocked" : "Active" },
        ]}
        actions={
          <>
            <button className="btn btn-sm" onClick={() => navigate("/users")}>
              Back
            </button>
            <button
              className="btn btn-sm"
              onClick={() => fileInputRef.current?.click()}
              disabled={uploadingAvatar}
            >
              {uploadingAvatar ? "Uploading..." : "Upload avatar"}
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              style={{ display: "none" }}
              onChange={handleAvatarChange}
            />
          </>
        }
      />

      <div className="detail-section">
        <div className="section-head">
          <h3>Account</h3>
        </div>
        <InfoGroups>
          <InfoGroup title="Access">
            <InfoRow label="Role">
              <select
                value={user.role}
                onChange={(e) => handleRoleChange(e.target.value as UserRole)}
              >
                <option value="guest">Guest</option>
                <option value="player">Player</option>
                <option value="admin">Admin</option>
              </select>
            </InfoRow>
            <InfoRow label="Blocked">
              {user.is_blocked ? "Yes" : "No"}
              {user.id !== currentUser?.id && (
                <button
                  className={`btn btn-sm ${user.is_blocked ? "" : "btn-danger"}`}
                  style={{ marginLeft: "0.5rem" }}
                  onClick={handleToggleBlock}
                >
                  {user.is_blocked ? "Unblock" : "Block"}
                </button>
              )}
            </InfoRow>
          </InfoGroup>
          <InfoGroup title="Meta">
            <InfoRow label="Created">{formatDateTime(user.created_at)}</InfoRow>
            <InfoRow label="Updated">{formatDateTime(user.updated_at)}</InfoRow>
          </InfoGroup>
        </InfoGroups>
      </div>

      <div className="detail-section">
        <div className="section-head">
          <h3>Profile</h3>
        </div>
        <form onSubmit={handleSaveProfile} className="edit-form">
          <div className="form-group">
            <label>Display Name</label>
            <input
              value={form.display_name}
              onChange={(e) =>
                setForm((f) => ({ ...f, display_name: e.target.value }))
              }
              required
            />
          </div>
          <div className="form-group">
            <label>About</label>
            <textarea
              value={form.bio}
              onChange={(e) => setForm((f) => ({ ...f, bio: e.target.value }))}
            />
          </div>
          <div className="form-group">
            <label>Telegram</label>
            <input
              value={form.telegram}
              onChange={(e) =>
                setForm((f) => ({ ...f, telegram: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>GitHub</label>
            <input
              value={form.github}
              onChange={(e) =>
                setForm((f) => ({ ...f, github: e.target.value }))
              }
            />
          </div>
          <div className="form-group">
            <label>Email</label>
            <input
              type="email"
              value={form.email}
              onChange={(e) =>
                setForm((f) => ({ ...f, email: e.target.value }))
              }
            />
          </div>
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {saving ? "Saving..." : "Save profile"}
            </button>
          </div>
        </form>
      </div>

      <div className="detail-section">
        <div className="section-head">
          <h3>Password</h3>
        </div>
        <form onSubmit={handleChangePassword} className="edit-form">
          <p className="section-hint">
            Setting a new password takes effect immediately. It does not affect
            the rest of the profile.
          </p>
          <div className="form-group">
            <label>New Password</label>
            <input
              type="password"
              autoComplete="new-password"
              minLength={6}
              value={passwordForm.password}
              onChange={(e) =>
                setPasswordForm((f) => ({ ...f, password: e.target.value }))
              }
              required
            />
          </div>
          <div className="form-group">
            <label>Confirm Password</label>
            <input
              type="password"
              autoComplete="new-password"
              minLength={6}
              value={passwordForm.confirm}
              onChange={(e) =>
                setPasswordForm((f) => ({ ...f, confirm: e.target.value }))
              }
              required
            />
          </div>
          <div className="form-actions">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={
                changingPassword ||
                !passwordForm.password ||
                !passwordForm.confirm
              }
            >
              {changingPassword ? "Updating..." : "Change password"}
            </button>
          </div>
        </form>
      </div>

      <div className="detail-section">
        <div className="section-head">
          <h3>
            Active Sessions <SectionCount n={sessions.length} />
          </h3>
        </div>
        {sessions.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th>IP address</th>
                <th>Client</th>
                <th>Last seen</th>
                <th>Started</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {sessions.map((s) => (
                <tr key={s.id}>
                  <td>
                    {s.ip_address ?? "—"}
                    {s.current && (
                      <span className="badge" style={{ marginLeft: "0.5rem" }}>
                        current
                      </span>
                    )}
                  </td>
                  <td title={s.user_agent ?? ""}>
                    {shortUserAgent(s.user_agent)}
                  </td>
                  <td>{formatRelativeTime(s.last_seen_at)}</td>
                  <td>{formatDateTime(s.created_at)}</td>
                  <td>
                    <ActionButton
                      onClick={() => handleRevokeSession(s.id)}
                      variant="danger"
                      confirm="Revoke this session?"
                    >
                      Revoke
                    </ActionButton>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="section-empty">No active sessions.</p>
        )}
      </div>

      <div className="detail-section">
        <div className="section-head">
          <h3>Danger Zone</h3>
        </div>
        {user.id === currentUser?.id ? (
          <p className="section-empty">You cannot delete your own account.</p>
        ) : !deleteOpen ? (
          <button
            className="btn btn-danger"
            onClick={() => setDeleteOpen(true)}
          >
            Delete user
          </button>
        ) : (
          <form onSubmit={handleDelete} className="edit-form">
            <p>
              This permanently deletes <strong>@{user.user_name}</strong> and
              all references to them. Confirm with your admin password.
            </p>
            <div className="form-group">
              <label>Your password</label>
              <input
                type="password"
                value={deletePassword}
                onChange={(e) => setDeletePassword(e.target.value)}
                required
              />
            </div>
            <div className="form-actions">
              <button
                type="submit"
                className="btn btn-danger"
                disabled={deleting || !deletePassword}
              >
                {deleting ? "Deleting..." : "Confirm delete"}
              </button>
              <button
                type="button"
                className="btn"
                onClick={() => {
                  setDeleteOpen(false);
                  setDeletePassword("");
                }}
              >
                Cancel
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

function shortUserAgent(ua?: string | null): string {
  if (!ua) return "—";
  return ua.length > 40 ? `${ua.slice(0, 40)}…` : ua;
}
