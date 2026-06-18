import { useState, useEffect, type ReactNode } from "react";

/** Shared building blocks for entity detail pages (hero + grouped info). */

export function DetailHero({
  kicker,
  title,
  badges,
  summary,
  actions,
  avatarUrl,
  avatarText,
}: {
  kicker: ReactNode;
  title: string;
  badges?: ReactNode;
  summary?: { label: string; value: ReactNode }[];
  actions?: ReactNode;
  avatarUrl?: string | null;
  avatarText?: string;
}) {
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    setFailed(false);
  }, [avatarUrl]);

  const hasLogo = Boolean(avatarUrl && !failed);
  const initial = (avatarText ?? title).trim().charAt(0).toUpperCase() || "?";

  return (
    <section className="detail-hero">
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

export function InfoGroups({ children }: { children: ReactNode }) {
  return <div className="info-groups">{children}</div>;
}

export function InfoGroup({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="info-group">
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
    return <em className="muted-dash">embedded image</em>;
  return renderLink(url);
}

export function formatDateTime(value?: string | null): string {
  return value ? new Date(value).toLocaleString() : "—";
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
      const label = `${n} ${name}${n === 1 ? "" : "s"}`;
      return future ? `in ${label}` : `${label} ago`;
    }
  }
  return "just now";
}

/** Full date/time including the timezone, e.g. for tooltips. */
export function formatDateTimeWithZone(value: string): string {
  return new Date(value).toLocaleString(undefined, {
    dateStyle: "full",
    timeStyle: "long",
  });
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
