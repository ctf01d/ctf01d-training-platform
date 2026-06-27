import { useState, useCallback, useEffect, useRef } from "react";
import * as usersApi from "../api/users";
import type { User, UserSession } from "../api/users";
import { ErrorDisplay, handleApiError } from "../components/ErrorDisplay";
import { usePageTitle } from "../components/usePageTitle";
import { useAuth } from "../auth/AuthContext";
import { useI18n } from "../i18n/I18nContext";
import {
  InfoGroups,
  InfoGroup,
  InfoRow,
  SectionCount,
  formatDateTime,
} from "../components/DetailInfo";
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
import { DEFAULT_THEME, setTheme, type ThemeId } from "../theme";

export default function ProfilePage() {
  const { user, refreshUser } = useAuth();
  const { t, roleLabel, languageLabel, setLanguage } = useI18n();
  usePageTitle(user?.display_name ?? t("Profile"));

  const [form, setForm] = useState<UserProfileFormState>(
    emptyUserProfileForm(),
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<{ message?: string } | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const [passwordForm, setPasswordForm] = useState<UserPasswordFormState>({
    password: "",
    confirm: "",
  });
  const [changingPassword, setChangingPassword] = useState(false);

  const [sessions, setSessions] = useState<UserSession[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);

  const [theme, setThemeSelection] = useState<ThemeId>(
    user?.theme ?? DEFAULT_THEME,
  );
  const [savingTheme, setSavingTheme] = useState(false);

  useEffect(() => {
    if (user) setForm(userProfileFormFromUser(user));
  }, [user]);

  useEffect(() => {
    if (user?.theme) setThemeSelection(user.theme);
  }, [user?.theme]);

  // Picking a theme applies it to the current UI immediately and persists it to
  // the profile. The full saved profile is resent so unrelated optional fields
  // (which the update endpoint clears when omitted) are preserved.
  const handleThemeChange = async (next: ThemeId) => {
    if (!user || next === theme) return;
    const previous = theme;
    setThemeSelection(next);
    setTheme(next);
    setSavingTheme(true);
    setError(null);
    setSuccess(null);
    const body = {
      ...profileUpdateFromForm(userProfileFormFromUser(user)),
      theme: next,
    };
    const { error: err } = await usersApi.updateProfile(body);
    setSavingTheme(false);
    if (err) {
      setError(err);
      setThemeSelection(previous);
      setTheme(previous);
      return;
    }
    await refreshUser();
  };

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

    const body = profileUpdateFromForm(form);

    const { data, error: err } = await usersApi.updateProfile(body);
    setSaving(false);
    if (err) {
      setError(err);
      return;
    }
    if (data) {
      setForm(userProfileFormFromUser(data));
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
      setForm(userProfileFormFromUser(data));
      await refreshUser();
      setSuccess("Avatar updated successfully.");
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setUploadingAvatar(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  if (!user) return <div className="loading">{t("Loading...")}</div>;

  return (
    <div className="page detail-page">
      <ErrorDisplay error={error} />
      {success && <div className="success-message">{t(success)}</div>}

      <UserDetailHero
        user={user}
        actions={
          <>
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
            <InfoRow label={t("Role")}>{roleLabel(user.role)}</InfoRow>
            <InfoRow label={t("Rating")}>{user.rating}</InfoRow>
            <InfoRow label={t("Status")}>
              {user.is_blocked ? t("Blocked") : t("Active")}
            </InfoRow>
          </InfoGroup>
          <InfoGroup title={t("Login")}>
            <InfoRow label={t("Last IP")}>{user.last_login_ip ?? "—"}</InfoRow>
            <InfoRow label={t("Last login")}>
              {user.last_login_at ? formatDateTime(user.last_login_at) : "—"}
            </InfoRow>
            <InfoRow label={t("Active sessions")}>{sessions.length}</InfoRow>
          </InfoGroup>
          <InfoGroup title={t("Profile")}>
            <InfoRow label={t("About")}>{user.bio || "—"}</InfoRow>
            <InfoRow label={t("Telegram")}>{user.telegram || "—"}</InfoRow>
            <InfoRow label={t("GitHub")}>{user.github || "—"}</InfoRow>
            <InfoRow label={t("Email")}>{user.email || "—"}</InfoRow>
            <InfoRow label={t("Language")}>
              {languageLabel(user.language)}
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
          onSubmit={handleSave}
          saving={saving}
          onLanguageChange={setLanguage}
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
          <button
            type="button"
            className="btn btn-sm"
            onClick={fetchSessions}
            disabled={sessionsLoading}
          >
            {sessionsLoading ? t("Refreshing...") : t("Refresh")}
          </button>
        </div>
        {sessionsLoading ? (
          <div className="loading">{t("Loading...")}</div>
        ) : (
          <UserSessionsTable sessions={sessions} showExpires />
        )}
      </div>
    </div>
  );
}
