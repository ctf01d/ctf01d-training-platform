import type { ReactNode } from "react";

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
  if (!error) return null;

  const fieldErrors =
    error.details && typeof error.details === "object"
      ? Object.entries(error.details).map(([field, msg]) => (
          <div key={field} className="field-error">
            <strong>{field}:</strong> {String(msg)}
          </div>
        ))
      : null;

  return (
    <div className="error-display">
      <p>{error.message ?? error.code ?? "An error occurred"}</p>
      {fieldErrors && <div className="field-errors">{fieldErrors}</div>}
      {onRetry && (
        <button onClick={onRetry} className="btn btn-sm">
          Retry
        </button>
      )}
    </div>
  );
}

export function ForbiddenDisplay() {
  return (
    <div className="error-display">
      You do not have permission to access this resource.
    </div>
  );
}

export function ErrorBoundary({ error }: { error: Error }) {
  return (
    <div className="error-display">
      <h3>Something went wrong</h3>
      <p>{error.message}</p>
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
