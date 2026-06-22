import { useState, useCallback, useEffect, useRef } from "react";
import * as usersApi from "../api/users";
import type { User, UserSession } from "../api/users";
import { ErrorDisplay, handleApiError } from "../components/ErrorDisplay";
import { usePageTitle } from "../components/usePageTitle";
import { useAuth } from "../auth/AuthContext";
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
import {
  emptyUserProfileForm,
  profileUpdateFromForm,
  userProfileFormFromUser,
  type UserPasswordFormState,
  type UserProfileFormState,
} from "../components/UserProfileModel";

export default function ProfilePage() {
  const { user, refreshUser } = useAuth();
  usePageTitle(user?.display_name ?? "Profile");

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

  useEffect(() => {
    if (user) setForm(userProfileFormFromUser(user));
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

  if (!user) return <div className="loading">Loading...</div>;

  return (
    <div className="page detail-page">
      <ErrorDisplay error={error} />
      {success && <div className="success-message">{success}</div>}

      <UserDetailHero
        user={user}
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
        <UserProfileEditForm
          form={form}
          setForm={setForm}
          onSubmit={handleSave}
          saving={saving}
        />
      </div>

      <div className="detail-section">
        <div className="section-head">
          <h3>Password</h3>
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
        ) : (
          <UserSessionsTable sessions={sessions} showExpires />
        )}
      </div>
    </div>
  );
}
