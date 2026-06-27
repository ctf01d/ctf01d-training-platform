import { useState, useEffect, useCallback, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import * as usersApi from "../api/users";
import type { User, UserSession, UserRole } from "../api/users";
import { ErrorDisplay, handleApiError } from "../components/ErrorDisplay";
import { usePageTitle } from "../components/usePageTitle";
import {
  InfoGroups,
  InfoGroup,
  InfoRow,
  SectionCount,
  formatDateTime,
} from "../components/DetailInfo";
import { useAuth } from "../auth/AuthContext";
import {
  UserDetailHero,
  UserPasswordForm,
  UserProfileEditForm,
  UserSessionsTable,
} from "../components/UserProfileBlocks";
import ThemeSection from "../components/ThemeSection";
import {
  emptyUserProfileForm,
  profileUpdateFromForm,
  userProfileFormFromUser,
  type UserPasswordFormState,
  type UserProfileFormState,
} from "../components/UserProfileModel";
import { DEFAULT_THEME, type ThemeId } from "../theme";
import { useI18n } from "../i18n/I18nContext";

export default function UserDetailPage() {
  const { t, roleLabel } = useI18n();
  const { id } = useParams<{ id: string }>();
  const userId = Number(id);
  const navigate = useNavigate();
  const { user: currentUser } = useAuth();

  const [user, setUser] = useState<User | null>(null);
  usePageTitle(user?.display_name ?? undefined);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ message?: string } | null>(null);

  const [form, setForm] = useState<UserProfileFormState>(
    emptyUserProfileForm(),
  );
  const [saving, setSaving] = useState(false);

  const [passwordForm, setPasswordForm] = useState<UserPasswordFormState>({
    password: "",
    confirm: "",
  });
  const [changingPassword, setChangingPassword] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);

  const [sessions, setSessions] = useState<UserSession[]>([]);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleting, setDeleting] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);

  const [theme, setThemeSelection] = useState<ThemeId>(DEFAULT_THEME);
  const [savingTheme, setSavingTheme] = useState(false);

  const applyUser = useCallback((u: User) => {
    setUser(u);
    setForm(userProfileFormFromUser(u));
    setThemeSelection(u.theme ?? DEFAULT_THEME);
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
    const body = profileUpdateFromForm(form);
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
      setSuccess(t("Profile updated successfully."));
    }
  };

  // Persists the viewed user's theme. This is their stored preference, so it is
  // not applied to the admin's own UI. The full saved profile is resent so the
  // update endpoint does not clear unrelated optional fields.
  const handleThemeChange = async (next: ThemeId) => {
    if (!user || next === theme) return;
    const previous = theme;
    setThemeSelection(next);
    setSavingTheme(true);
    setError(null);
    setSuccess(null);
    const body = {
      ...profileUpdateFromForm(userProfileFormFromUser(user)),
      theme: next,
    };
    const { data, error: err } = await usersApi.updateUserProfileAdmin(
      userId,
      body,
    );
    setSavingTheme(false);
    if (err) {
      setError(err);
      setThemeSelection(previous);
      return;
    }
    if (data) {
      applyUser(data);
      setSuccess(t("Profile updated successfully."));
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
    setSuccess(t("Password updated successfully."));
  };

  const handleRoleChange = async (role: UserRole) => {
    if (userId === currentUser?.id) return;
    setError(null);
    setSuccess(null);
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
    setSuccess(null);
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

  if (loading) return <div className="loading">{t("Loading...")}</div>;
  if (!user) return <ErrorDisplay error={error} onRetry={fetchUser} />;

  return (
    <div className="page detail-page">
      <ErrorDisplay error={error} onRetry={fetchUser} />
      {success && <div className="success-message">{success}</div>}

      <UserDetailHero
        user={user}
        actions={
          <>
            <button className="btn btn-sm" onClick={() => navigate("/users")}>
              {t("Back")}
            </button>
            <button
              className="btn btn-sm"
              onClick={() => fileInputRef.current?.click()}
              disabled={uploadingAvatar}
            >
              {uploadingAvatar ? t("Uploading...") : t("Upload avatar")}
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
          <h3>{t("Account")}</h3>
        </div>
        <InfoGroups>
          <InfoGroup title={t("Access")}>
            <InfoRow label={t("Role")}>
              <select
                value={user.role}
                onChange={(e) => handleRoleChange(e.target.value as UserRole)}
                disabled={user.id === currentUser?.id}
              >
                <option value="guest">{roleLabel("guest")}</option>
                <option value="player">{roleLabel("player")}</option>
                <option value="admin">{roleLabel("admin")}</option>
              </select>
              {user.id === currentUser?.id && (
                <span className="section-hint" style={{ marginLeft: "0.5rem" }}>
                  {t("You cannot change your own role.")}
                </span>
              )}
            </InfoRow>
            <InfoRow label={t("Blocked")}>
              {user.is_blocked ? t("Yes") : t("No")}
              {user.id !== currentUser?.id && (
                <button
                  className={`btn btn-sm ${user.is_blocked ? "" : "btn-danger"}`}
                  style={{ marginLeft: "0.5rem" }}
                  onClick={handleToggleBlock}
                >
                  {user.is_blocked ? t("Unblock") : t("Block")}
                </button>
              )}
            </InfoRow>
          </InfoGroup>
          <InfoGroup title={t("Meta")}>
            <InfoRow label={t("Created")}>
              {formatDateTime(user.created_at)}
            </InfoRow>
            <InfoRow label={t("Updated")}>
              {formatDateTime(user.updated_at)}
            </InfoRow>
          </InfoGroup>
        </InfoGroups>
      </div>

      <ThemeSection
        value={theme}
        onChange={handleThemeChange}
        disabled={savingTheme}
      />

      <div className="detail-section">
        <div className="section-head">
          <h3>{t("Profile")}</h3>
        </div>
        <UserProfileEditForm
          form={form}
          setForm={setForm}
          onSubmit={handleSaveProfile}
          saving={saving}
        />
      </div>

      <div className="detail-section">
        <div className="section-head">
          <h3>{t("Password")}</h3>
        </div>
        <UserPasswordForm
          form={passwordForm}
          setForm={setPasswordForm}
          onSubmit={handleChangePassword}
          changing={changingPassword}
        />
      </div>

      <div className="detail-section">
        <div className="section-head">
          <h3>
            {t("Active sessions")} <SectionCount n={sessions.length} />
          </h3>
        </div>
        <UserSessionsTable sessions={sessions} onRevoke={handleRevokeSession} />
      </div>

      <div className="detail-section">
        <div className="section-head">
          <h3>{t("Danger Zone")}</h3>
        </div>
        {user.id === currentUser?.id ? (
          <p className="section-empty">
            {t("You cannot delete your own account.")}
          </p>
        ) : !deleteOpen ? (
          <button
            className="btn btn-danger"
            onClick={() => setDeleteOpen(true)}
          >
            {t("Delete user")}
          </button>
        ) : (
          <form onSubmit={handleDelete} className="edit-form">
            <p>
              {t(
                "This permanently deletes @{username} and all references to them. Confirm with your admin password.",
                { username: user.user_name },
              )}
            </p>
            <div className="form-group">
              <label>{t("Your password")}</label>
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
                {deleting ? t("Deleting...") : t("Confirm delete")}
              </button>
              <button
                type="button"
                className="btn"
                onClick={() => {
                  setDeleteOpen(false);
                  setDeletePassword("");
                }}
              >
                {t("Cancel")}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
