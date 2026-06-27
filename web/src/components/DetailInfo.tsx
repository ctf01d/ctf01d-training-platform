import { useState, useEffect, type ReactNode } from "react";
import {
  formatRelativeLiteral,
  getCurrentLanguage,
  localeForLanguage,
  translateCurrent,
} from "../i18n/runtime";

/** Shared building blocks for entity detail pages (hero + grouped info). */

export function DetailHero({
  kicker,
  title,
  badges,
  summary,
  actions,
  avatarUrl,
  avatarText,
  avatarMode = "logo",
}: {
  kicker: ReactNode;
  title: string;
  badges?: ReactNode;
  summary?: { label: string; value: ReactNode }[];
  actions?: ReactNode;
  avatarUrl?: string | null;
  avatarText?: string;
  avatarMode?: "logo" | "photo";
}) {
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    setFailed(false);
  }, [avatarUrl]);

  const hasLogo = Boolean(avatarUrl && !failed);
  const initial = (avatarText ?? title).trim().charAt(0).toUpperCase() || "?";

  return (
    <section
      className={
        avatarMode === "photo"
          ? "detail-hero detail-hero--photo"
          : "detail-hero"
      }
    >
      <div className="detail-hero-content">
        <div className="detail-hero-kicker">{kicker}</div>
        <h1>{title}</h1>
        {badges && <div className="detail-hero-badges">{badges}</div>}
        {summary && summary.length > 0 && (
          <div className="detail-hero-summary">
            {summary.map((s) => (
              <div key={s.label}>
                <span>{s.label}</span>
                <strong>{s.value}</strong>
              </div>
            ))}
          </div>
        )}
        {actions && <div className="detail-hero-links">{actions}</div>}
      </div>

      <div className="detail-hero-logo" aria-hidden="true">
        {hasLogo ? (
          <img src={avatarUrl ?? ""} alt="" onError={() => setFailed(true)} />
        ) : (
          <span>{initial}</span>
        )}
      </div>
    </section>
  );
}

export function InfoGroups({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={className ? `info-groups ${className}` : "info-groups"}>
      {children}
    </div>
  );
}

export function InfoGroup({
  title,
  children,
  className,
}: {
  title: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={className ? `info-group ${className}` : "info-group"}>
      <h4>{title}</h4>
      <dl className="info-dl">{children}</dl>
    </div>
  );
}

export function InfoRow({
  label,
  children,
}: {
  label: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="info-row">
      <dt>{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

export function SectionCount({ n }: { n: number }) {
  return <span className="section-count">{n}</span>;
}

export function renderLink(url?: string | null): ReactNode {
  if (!url) return <span className="muted-dash">—</span>;
  return (
    <a href={safeHref(url)} target="_blank" rel="noreferrer" title={url}>
      {url.replace(/^https?:\/\//, "").replace(/\/$/, "")}
    </a>
  );
}

export function renderLogo(url?: string | null): ReactNode {
  if (!url) return <span className="muted-dash">—</span>;
  if (url.startsWith("data:"))
    return <em className="muted-dash">{translateCurrent("embedded image")}</em>;
  return renderLink(url);
}

export function formatDateTime(value?: string | null): string {
  return value
    ? new Date(value).toLocaleString(localeForLanguage(getCurrentLanguage()))
    : "—";
}

/** GitHub-style relative phrasing: "4 days ago", "in 8 hours", "just now". */
export function formatRelativeTime(value?: string | null): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  const diffMs = date.getTime() - Date.now();
  const future = diffMs > 0;
  const abs = Math.abs(diffMs);
  const units: [number, string][] = [
    [365 * 24 * 3600e3, "year"],
    [30 * 24 * 3600e3, "month"],
    [7 * 24 * 3600e3, "week"],
    [24 * 3600e3, "day"],
    [3600e3, "hour"],
    [60e3, "minute"],
    [1e3, "second"],
  ];
  for (const [ms, name] of units) {
    const n = Math.floor(abs / ms);
    if (n >= 1) {
      return formatRelativeLiteral(n, name, future);
    }
  }
  return translateCurrent("just now");
}

/**
 * Human-readable span between two timestamps, e.g. "7h", "1d 4h", "45m".
 * Returns null when either bound is missing or the range is non-positive.
 */
export function formatDuration(
  start?: string | null,
  end?: string | null,
): string | null {
  if (!start || !end) return null;
  const ms = new Date(end).getTime() - new Date(start).getTime();
  if (!Number.isFinite(ms) || ms <= 0) return null;
  const totalMinutes = Math.round(ms / 60000);
  const days = Math.floor(totalMinutes / 1440);
  const hours = Math.floor((totalMinutes % 1440) / 60);
  const minutes = totalMinutes % 60;
  const parts: string[] = [];
  if (days) parts.push(`${days}${getCurrentLanguage() === "ru" ? "д" : "d"}`);
  if (hours) parts.push(`${hours}${getCurrentLanguage() === "ru" ? "ч" : "h"}`);
  if (minutes && !days)
    parts.push(`${minutes}${getCurrentLanguage() === "ru" ? "м" : "m"}`);
  return parts.join(" ") || (getCurrentLanguage() === "ru" ? "0м" : "0m");
}

/**
 * Renders the span between two timestamps as a human-readable duration,
 * e.g. "1d 4h". Falls back to a muted dash when the range is unknown.
 */
export function Duration({
  start,
  end,
}: {
  start?: string | null;
  end?: string | null;
}) {
  const text = formatDuration(start, end);
  if (!text) return <span className="muted-dash">—</span>;
  return <span>{text}</span>;
}

/** Full date/time including the timezone, e.g. for tooltips. */
export function formatDateTimeWithZone(value: string): string {
  return new Date(value).toLocaleString(
    localeForLanguage(getCurrentLanguage()),
    {
      dateStyle: "full",
      timeStyle: "long",
    },
  );
}

/**
 * Renders a timestamp GitHub-style: relative text in the page, exact date,
 * time and timezone shown on hover.
 */
export function RelativeTime({ value }: { value?: string | null }) {
  if (!value) return <span className="muted-dash">—</span>;
  return (
    <time dateTime={value} title={formatDateTimeWithZone(value)}>
      {formatRelativeTime(value)}
    </time>
  );
}

export function safeHref(url: string): string {
  if (/^https?:\/\//i.test(url)) return url;
  return "about:blank";
}
