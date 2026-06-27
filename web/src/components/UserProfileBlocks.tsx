import type { Dispatch, FormEvent, ReactNode, SetStateAction } from "react";
import type { User, UserSession } from "../api/users";
import { useI18n } from "../i18n/I18nContext";
import { ActionButton } from "./ErrorDisplay";
import { DetailHero, formatDateTime, formatRelativeTime } from "./DetailInfo";
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
  const { t, roleLabel } = useI18n();
  const title = userTitle(user);

  return (
    <DetailHero
      kicker={`${t("User")} #${user.id}`}
      title={title}
      avatarUrl={user.avatar_url}
      avatarText={title}
      avatarMode="photo"
      badges={
        <span className={`badge badge-${user.role}`}>
          {roleLabel(user.role)}
        </span>
      }
      summary={[
        { label: t("Username"), value: `@${user.user_name}` },
        { label: t("Role"), value: roleLabel(user.role) },
        { label: t("Rating"), value: `${user.rating}` },
        {
          label: t("Status"),
          value: user.is_blocked ? t("Blocked") : t("Active"),
        },
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
  onLanguageChange,
}: {
  form: UserProfileFormState;
  setForm: Dispatch<SetStateAction<UserProfileFormState>>;
  onSubmit: (e: FormEvent<HTMLFormElement>) => void;
  saving: boolean;
  onLanguageChange?: (language: User["language"]) => void;
}) {
  const { t, languageLabel } = useI18n();
  return (
    <form onSubmit={onSubmit} className="edit-form" autoComplete="off">
      <div className="form-group">
        <label>{t("Display Name")}</label>
        <input
          value={form.display_name}
          onChange={(e) =>
            setForm((f) => ({ ...f, display_name: e.target.value }))
          }
          autoComplete="off"
          required
        />
      </div>
      <div className="form-group">
        <label>{t("About")}</label>
        <textarea
          value={form.bio}
          onChange={(e) => setForm((f) => ({ ...f, bio: e.target.value }))}
        />
      </div>
      <div className="form-group">
        <label>{t("Telegram")}</label>
        <input
          value={form.telegram}
          onChange={(e) => setForm((f) => ({ ...f, telegram: e.target.value }))}
        />
      </div>
      <div className="form-group">
        <label>{t("GitHub")}</label>
        <input
          value={form.github}
          onChange={(e) => setForm((f) => ({ ...f, github: e.target.value }))}
        />
      </div>
      <div className="form-group">
        <label>{t("Email")}</label>
        <input
          type="email"
          value={form.email}
          onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
          autoComplete="off"
        />
      </div>
      <div className="form-group">
        <label>{t("Language")}</label>
        <select
          value={form.language}
          onChange={(e) => {
            const language = e.target.value as User["language"];
            setForm((f) => ({
              ...f,
              language,
            }));
            onLanguageChange?.(language);
          }}
        >
          <option value="en">{`🇺🇸 ${languageLabel("en")}`}</option>
          <option value="ru">{`🇷🇺 ${languageLabel("ru")}`}</option>
        </select>
      </div>
      <div className="form-actions">
        <button type="submit" className="btn btn-primary" disabled={saving}>
          {saving ? t("Saving...") : t("Save profile")}
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
  const { t } = useI18n();
  return (
    <form onSubmit={onSubmit} className="edit-form password-form">
      <p className="section-hint">
        {t(
          "Setting a new password takes effect immediately. It does not affect the rest of the profile.",
        )}
      </p>
      <div className="form-group">
        <label>{t("New Password")}</label>
        <input
          type="password"
          autoComplete="new-password"
          minLength={6}
          value={form.password}
          onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
          required
        />
      </div>
      <div className="form-group">
        <label>{t("Confirm Password")}</label>
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
          {changing ? t("Updating...") : t("Change password")}
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
  const { t } = useI18n();
  if (sessions.length === 0) {
    return <p className="section-empty">{t("No active sessions.")}</p>;
  }

  return (
    <table className="data-table">
      <thead>
        <tr>
          <th>{t("IP address")}</th>
          <th>{t("Client")}</th>
          <th>{t("Last seen")}</th>
          <th>{t("Started")}</th>
          {showExpires && <th>{t("Expires")}</th>}
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
                  {t("current")}
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
                  confirm={t("Revoke this session?")}
                >
                  {t("Revoke")}
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
