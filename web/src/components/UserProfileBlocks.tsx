import type {
  Dispatch,
  FormEvent,
  ReactNode,
  SetStateAction,
} from "react";
import type { User, UserSession } from "../api/users";
import { ActionButton } from "./ErrorDisplay";
import {
  DetailHero,
  formatDateTime,
  formatRelativeTime,
} from "./DetailInfo";
import {
  userTitle,
  type UserPasswordFormState,
  type UserProfileFormState,
} from "./UserProfileModel";

export function UserDetailHero({
  user,
  actions,
}: {
  user: User;
  actions?: ReactNode;
}) {
  const title = userTitle(user);

  return (
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
        { label: "Status", value: user.is_blocked ? "Blocked" : "Active" },
      ]}
      actions={actions}
    />
  );
}

export function UserProfileEditForm({
  form,
  setForm,
  onSubmit,
  saving,
}: {
  form: UserProfileFormState;
  setForm: Dispatch<SetStateAction<UserProfileFormState>>;
  onSubmit: (e: FormEvent<HTMLFormElement>) => void;
  saving: boolean;
}) {
  return (
    <form onSubmit={onSubmit} className="edit-form">
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
          onChange={(e) => setForm((f) => ({ ...f, github: e.target.value }))}
        />
      </div>
      <div className="form-group">
        <label>Email</label>
        <input
          type="email"
          value={form.email}
          onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
        />
      </div>
      <div className="form-actions">
        <button type="submit" className="btn btn-primary" disabled={saving}>
          {saving ? "Saving..." : "Save profile"}
        </button>
      </div>
    </form>
  );
}

export function UserPasswordForm({
  form,
  setForm,
  onSubmit,
  changing,
}: {
  form: UserPasswordFormState;
  setForm: Dispatch<SetStateAction<UserPasswordFormState>>;
  onSubmit: (e: FormEvent<HTMLFormElement>) => void;
  changing: boolean;
}) {
  return (
    <form onSubmit={onSubmit} className="edit-form password-form">
      <p className="section-hint">
        Setting a new password takes effect immediately. It does not affect the
        rest of the profile.
      </p>
      <div className="form-group">
        <label>New Password</label>
        <input
          type="password"
          autoComplete="new-password"
          minLength={6}
          value={form.password}
          onChange={(e) =>
            setForm((f) => ({ ...f, password: e.target.value }))
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
          value={form.confirm}
          onChange={(e) => setForm((f) => ({ ...f, confirm: e.target.value }))}
          required
        />
      </div>
      <div className="form-actions">
        <button
          type="submit"
          className="btn btn-primary"
          disabled={changing || !form.password || !form.confirm}
        >
          {changing ? "Updating..." : "Change password"}
        </button>
      </div>
    </form>
  );
}

export function UserSessionsTable({
  sessions,
  showExpires = false,
  onRevoke,
}: {
  sessions: UserSession[];
  showExpires?: boolean;
  onRevoke?: (sessionId: number) => void;
}) {
  if (sessions.length === 0) {
    return <p className="section-empty">No active sessions.</p>;
  }

  return (
    <table className="data-table">
      <thead>
        <tr>
          <th>IP address</th>
          <th>Client</th>
          <th>Last seen</th>
          <th>Started</th>
          {showExpires && <th>Expires</th>}
          {onRevoke && <th></th>}
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
            <td title={s.user_agent ?? ""}>{shortUserAgent(s.user_agent)}</td>
            <td>{formatRelativeTime(s.last_seen_at)}</td>
            <td>{formatDateTime(s.created_at)}</td>
            {showExpires && <td>{formatDateTime(s.expires_at)}</td>}
            {onRevoke && (
              <td>
                <ActionButton
                  onClick={() => onRevoke(s.id)}
                  variant="danger"
                  confirm="Revoke this session?"
                >
                  Revoke
                </ActionButton>
              </td>
            )}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function shortUserAgent(ua?: string | null): string {
  if (!ua) return "—";
  return ua.length > 40 ? `${ua.slice(0, 40)}...` : ua;
}
