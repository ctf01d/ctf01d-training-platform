import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useI18n } from "../i18n/I18nContext";

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { t } = useI18n();
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) return <div className="loading">{t("Loading...")}</div>;
  if (!user) return <Navigate to="/login" state={{ from: location }} replace />;

  return <>{children}</>;
}

export function AdminRoute({ children }: { children: React.ReactNode }) {
  const { t } = useI18n();
  const { user, loading, isAdmin } = useAuth();
  const location = useLocation();

  if (loading) return <div className="loading">{t("Loading...")}</div>;
  if (!user) return <Navigate to="/login" state={{ from: location }} replace />;
  if (!isAdmin) return <Navigate to="/" replace />;

  return <>{children}</>;
}
