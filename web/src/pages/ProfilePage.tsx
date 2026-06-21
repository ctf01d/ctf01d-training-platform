import { useState, useCallback, useEffect, useRef } from "react";
import * as usersApi from "../api/users";
import type { User, UserProfileUpdate, UserSession } from "../api/users";
import { ErrorDisplay, handleApiError } from "../components/ErrorDisplay";
import { usePageTitle } from "../components/usePageTitle";
import { useAuth } from "../auth/AuthContext";
import {
  DetailHero,
  InfoGroups,
  InfoGroup,
  InfoRow,
  SectionCount,
  formatDateTime,
  formatRelativeTime,
} from "../components/DetailInfo";

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

function formFromUser(user: User): ProfileForm {
  return {
    display_name: user.display_name ?? "",
    bio: user.bio ?? "",
    telegram: user.telegram ?? "",
    github: user.github ?? "",
    email: user.email ?? "",
  };
}

export default function ProfilePage() {
  const { user, refreshUser } = useAuth();
  usePageTitle(user?.display_name ?? "Profile");

  const [form, setForm] = useState<ProfileForm>(emptyProfileForm());
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<{ message?: string } | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const [passwordForm, setPasswordForm] = useState({ password: "", confirm: "" });
  const [changingPassword, setChangingPassword] = useState(false);

  const [sessions, setSessions] = useState<UserSession[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);

  useEffect(() => {
    if (user) setForm(formFromUser(user));
  }, [user]);

  const fetchSessions = useCallback(async () => {
    setSessionsLoading(true);
    const { data, error: err } = await usersApi.listProfileSessions();
    if (!err && data) {
      setSessions(data.items);
    }
    setSessionsLoading(false);
  }, []);

  useEffect(() => {
    if (user) void fetchSessions();
  }, [user, fetchSessions]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(null);
    setSuccess(null);

    const body: UserProfileUpdate = {
      display_name: form.display_name,
      bio: form.bio || null,
      telegram: form.telegram || null,
      github: form.github || null,
      email: form.email || null,
    };

    const { data, error: err } = await usersApi.updateProfile(body);
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      setForm(formFromUser(data));
      await refreshUser();
      setSuccess("Profile updated successfully.");
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
    // Dedicated endpoint: only the password changes, so the open profile form
    // (including any unsaved edits) is left untouched.
    const { error: err } = await usersApi.changeProfilePassword(
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

  const handleAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploadingAvatar(true);
    setError(null);
    setSuccess(null);
    try {
      const response = await usersApi.uploadProfileAvatar(file);
      if (!response.ok) {
        const body = await response.json();
        setError(handleApiError(body));
        return;
      }
      const data = (await response.json()) as User;
      setForm(formFromUser(data));
      await refreshUser();
      setSuccess("Avatar updated successfully.");
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setUploadingAvatar(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  if (!user) return <div className="loading">Loading...</div>;

  const title = user.display_name || user.user_name;

  return (
    <div className="page detail-page">
      <ErrorDisplay error={error} />
      {success && <div className="success-message">{success}</div>}

      <DetailHero
        kicker={`User #${user.id}`}
        title={title}
        avatarUrl={user.avatar_url}
        avatarText={title}
        avatarMode="photo"
        badges={<span className={`badge badge-${user.role}`}>{user.role}</span>}
        summary={[
          { label: "Username", value: `@${user.user_name}` },
          { label: "Role", value: user.role },
          { label: "Rating", value: `${user.rating}` },
          { label: "Last login IP", value: user.last_login_ip ?? "—" },
        ]}
        actions={
          <>
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
            <InfoRow label="Role">{user.role}</InfoRow>
            <InfoRow label="Rating">{user.rating}</InfoRow>
            <InfoRow label="Status">
              {user.is_blocked ? "Blocked" : "Active"}
            </InfoRow>
          </InfoGroup>
          <InfoGroup title="Login">
            <InfoRow label="Last IP">{user.last_login_ip ?? "—"}</InfoRow>
            <InfoRow label="Last login">
              {user.last_login_at ? formatDateTime(user.last_login_at) : "—"}
            </InfoRow>
            <InfoRow label="Active sessions">{sessions.length}</InfoRow>
          </InfoGroup>
          <InfoGroup title="Profile">
            <InfoRow label="About">{user.bio || "—"}</InfoRow>
            <InfoRow label="Telegram">{user.telegram || "—"}</InfoRow>
            <InfoRow label="GitHub">{user.github || "—"}</InfoRow>
            <InfoRow label="Email">{user.email || "—"}</InfoRow>
          </InfoGroup>
        </InfoGroups>
      </div>

      <div className="detail-section">
        <div className="section-head">
          <h3>Profile</h3>
        </div>
        <form onSubmit={handleSave} className="edit-form">
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
            the rest of your profile.
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
          <button
            type="button"
            className="btn btn-sm"
            onClick={fetchSessions}
            disabled={sessionsLoading}
          >
            {sessionsLoading ? "Refreshing..." : "Refresh"}
          </button>
        </div>
        {sessionsLoading ? (
          <div className="loading">Loading...</div>
        ) : sessions.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th>IP address</th>
                <th>Client</th>
                <th>Last seen</th>
                <th>Started</th>
                <th>Expires</th>
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
                  <td>{formatDateTime(s.expires_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="section-empty">No active sessions.</p>
        )}
      </div>
    </div>
  );
}

function shortUserAgent(ua?: string | null): string {
  if (!ua) return "—";
  return ua.length > 40 ? `${ua.slice(0, 40)}...` : ua;
}
