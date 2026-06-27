import type { ReactNode } from "react";
import { useI18n } from "../i18n/I18nContext";

interface ApiError {
  code?: string;
  message?: string;
  details?: Record<string, unknown> | null;
}

interface ErrorDisplayProps {
  error: ApiError | null | undefined;
  onRetry?: () => void;
}

export function ErrorDisplay({ error, onRetry }: ErrorDisplayProps) {
  const { t } = useI18n();
  if (!error) return null;

  const fieldErrors =
    error.details && typeof error.details === "object"
      ? Object.entries(error.details).map(([field, msg]) => (
          <div key={field} className="field-error">
            <strong>{field}:</strong> {t(String(msg))}
          </div>
        ))
      : null;

  return (
    <div className="error-display">
      <p>{t(error.message ?? error.code ?? "An error occurred")}</p>
      {fieldErrors && <div className="field-errors">{fieldErrors}</div>}
      {onRetry && (
        <button onClick={onRetry} className="btn btn-sm">
          {t("Retry")}
        </button>
      )}
    </div>
  );
}

export function ForbiddenDisplay() {
  const { t } = useI18n();
  return (
    <div className="error-display">
      {t("You do not have permission to access this resource.")}
    </div>
  );
}

export function ErrorBoundary({ error }: { error: Error }) {
  const { t } = useI18n();
  return (
    <div className="error-display">
      <h3>{t("Something went wrong")}</h3>
      <p>{t(error.message)}</p>
    </div>
  );
}

export function handleApiError(err: unknown): ApiError {
  if (err && typeof err === "object" && "message" in err) {
    return err as ApiError;
  }
  return { message: String(err) };
}

export function ActionButton({
  onClick,
  children,
  variant = "default",
  disabled,
  confirm,
}: {
  onClick: () => void;
  children: ReactNode;
  variant?: "default" | "danger" | "success";
  disabled?: boolean;
  confirm?: string;
}) {
  const handleClick = () => {
    if (confirm && !window.confirm(confirm)) return;
    onClick();
  };
  const cls =
    variant === "danger"
      ? "btn btn-sm btn-danger"
      : variant === "success"
        ? "btn btn-sm btn-success"
        : "btn btn-sm";
  return (
    <button className={cls} onClick={handleClick} disabled={disabled}>
      {children}
    </button>
  );
}
