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
import {
  emptyUserProfileForm,
  profileUpdateFromForm,
  userProfileFormFromUser,
  type UserPasswordFormState,
  type UserProfileFormState,
} from "../components/UserProfileModel";

export default function UserDetailPage() {
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

  const applyUser = useCallback((u: User) => {
    setUser(u);
    setForm(userProfileFormFromUser(u));
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

  if (loading) return <div className="loading">Loading...</div>;
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
                disabled={user.id === currentUser?.id}
              >
                <option value="guest">Guest</option>
                <option value="player">Player</option>
                <option value="admin">Admin</option>
              </select>
              {user.id === currentUser?.id && (
                <span className="section-hint" style={{ marginLeft: "0.5rem" }}>
                  You cannot change your own role.
                </span>
              )}
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
        <UserProfileEditForm
          form={form}
          setForm={setForm}
          onSubmit={handleSaveProfile}
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
        </div>
        <UserSessionsTable sessions={sessions} onRevoke={handleRevokeSession} />
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
