import { type ReactNode } from 'react'
import { Link } from 'react-router-dom'

export function CardGrid({
  loading,
  isEmpty,
  emptyMessage = 'No data',
  children,
}: {
  loading?: boolean
  isEmpty?: boolean
  emptyMessage?: string
  children: ReactNode
}) {
  if (loading) return <div className="loading">Loading...</div>
  if (isEmpty) return <div className="empty-state">{emptyMessage}</div>
  return <div className="card-grid">{children}</div>
}

export function CardBadge({ variant, children }: { variant: string; children: ReactNode }) {
  return <span className={`badge badge-${variant}`}>{children}</span>
}

function CardAvatar({ url, text }: { url?: string | null; text?: string }) {
  const initial = (text ?? '?').trim().charAt(0).toUpperCase() || '?'
  if (url) return <img className="ecard-avatar" src={url} alt="" loading="lazy" />
  return <span className="ecard-avatar ecard-avatar--fallback">{initial}</span>
}

export function EntityCard({
  to,
  avatarUrl,
  avatarText,
  title,
  badges,
  children,
  footer,
}: {
  to?: string
  avatarUrl?: string | null
  avatarText?: string
  title: ReactNode
  badges?: ReactNode
  children?: ReactNode
  footer?: ReactNode
}) {
  return (
    <div className="entity-card">
      <div className="ecard-link">
        <div className="ecard-head">
          <CardAvatar url={avatarUrl} text={avatarText} />
          <div className="ecard-heading">
            <div className="ecard-title">
              {to ? (
                <Link to={to} className="ecard-stretched">
                  {title}
                </Link>
              ) : (
                title
              )}
            </div>
            {badges && <div className="ecard-badges">{badges}</div>}
          </div>
        </div>
        {children && <div className="ecard-body">{children}</div>}
      </div>
      {footer && <div className="ecard-footer">{footer}</div>}
    </div>
  )
}

export function CardMeta({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="ecard-meta">
      <span className="ecard-meta-label">{label}</span>
      <span className="ecard-meta-value">{children}</span>
    </div>
  )
}

export function Pagination({
  page,
  perPage,
  total,
  onPageChange,
}: {
  page: number
  perPage: number
  total: number
  onPageChange: (page: number) => void
}) {
  const totalPages = Math.ceil(total / perPage)
  if (totalPages <= 1) return null
  return (
    <div className="pagination">
      <button className="btn btn-sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
        Prev
      </button>
      <span>
        Page {page} of {totalPages} ({total} items)
      </span>
      <button className="btn btn-sm" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>
        Next
      </button>
    </div>
  )
}
