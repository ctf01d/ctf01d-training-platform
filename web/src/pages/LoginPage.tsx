import { useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { usePageTitle } from "../components/usePageTitle";
import { useAuth } from "../auth/AuthContext";
import { useI18n } from "../i18n/I18nContext";

export default function LoginPage() {
  const { t } = useI18n();
  usePageTitle(t("Login"));
  const [userName, setUserName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const from =
    (location.state as { from?: { pathname: string } } | undefined)?.from
      ?.pathname ?? "/";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(userName, password);
      navigate(from, { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("Login failed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page">
      <form onSubmit={handleSubmit} className="login-form">
        <h1>CTF01D Training Platform</h1>
        <h2>{t("Sign In")}</h2>
        {error && <div className="error-display">{t(error)}</div>}
        <div className="form-group">
          <label>{t("Username")}</label>
          <input
            type="text"
            value={userName}
            onChange={(e) => setUserName(e.target.value)}
            required
            autoFocus
            disabled={loading}
          />
        </div>
        <div className="form-group">
          <label>{t("Password")}</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            disabled={loading}
          />
        </div>
        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading ? t("Signing in...") : t("Sign In")}
        </button>
      </form>
    </div>
  );
}
