import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import * as gamesApi from '../api/games'
import type { Game, GameCreate } from '../api/games'
import { DataTable } from '../components/DataTable'
import { ErrorDisplay, ActionButton } from '../components/ErrorDisplay'
import { useAuth } from '../auth/AuthContext'

const statusColors: Record<string, string> = {
  upcoming: '#3b82f6',
  ongoing: '#22c55e',
  past: '#6b7280',
  unknown: '#9ca3af',
}

const regStatusColors: Record<string, string> = {
  unscheduled: '#9ca3af',
  upcoming: '#3b82f6',
  open: '#22c55e',
  closed: '#ef4444',
}

const sbStatusColors: Record<string, string> = {
  always: '#22c55e',
  upcoming: '#3b82f6',
  open: '#22c55e',
  closed: '#ef4444',
}

function Badge({ label, color }: { label: string; color: string }) {
  return (
    <span style={{ backgroundColor: color, color: '#fff', padding: '2px 8px', borderRadius: 999, fontSize: 12 }}>
      {label}
    </span>
  )
}

export default function GamesPage() {
  const { isPlayer } = useAuth()
  const navigate = useNavigate()
  const [games, setGames] = useState<Game[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<{ message?: string } | null>(null)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const perPage = 20
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState<GameCreate>({})
  const [creating, setCreating] = useState(false)

  const fetchGames = useCallback(async () => {
    setLoading(true)
    setError(null)
    const { data, error: err } = await gamesApi.listGames({ page, per_page: perPage })
    if (err) {
      setError(err)
    } else if (data) {
      setGames(data.items)
      setTotal(data.pagination.total)
    }
    setLoading(false)
  }, [page])

  useEffect(() => {
    void fetchGames()
  }, [fetchGames])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)
    const { data, error: err } = await gamesApi.createGame(form)
    setCreating(false)
    if (err) {
      setError(err)
      return
    }
    if (data) {
      navigate(`/games/${data.id}`)
    }
  }

  const columns = [
    { header: 'Name', render: (g: Game) => <a href={`/games/${g.id}`}>{g.name ?? '—'}</a> },
    { header: 'Organizer', render: (g: Game) => g.organizer ?? '—' },
    { header: 'Starts', render: (g: Game) => g.starts_at ? new Date(g.starts_at).toLocaleString() : '—' },
    { header: 'Ends', render: (g: Game) => g.ends_at ? new Date(g.ends_at).toLocaleString() : '—' },
    {
      header: 'Status',
      render: (g: Game) => <Badge label={g.status ?? 'unknown'} color={statusColors[g.status ?? 'unknown'] ?? '#9ca3af'} />,
    },
    {
      header: 'Registration',
      render: (g: Game) => <Badge label={g.registration_status ?? 'unscheduled'} color={regStatusColors[g.registration_status ?? 'unscheduled'] ?? '#9ca3af'} />,
    },
    {
      header: 'Scoreboard',
      render: (g: Game) => <Badge label={g.scoreboard_status ?? 'closed'} color={sbStatusColors[g.scoreboard_status ?? 'closed'] ?? '#9ca3af'} />,
    },
    {
      header: 'Finalized',
      render: (g: Game) => g.finalized ? 'Yes' : 'No',
    },
  ]

  return (
    <div className="page">
      <div className="page-header">
        <h1>Games</h1>
        {isPlayer && (
          <button className="btn btn-primary" onClick={() => setShowCreate(!showCreate)}>
            {showCreate ? 'Cancel' : 'Create Game'}
          </button>
        )}
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="create-form">
          <div className="form-group">
            <label>Name</label>
            <input value={form.name ?? ''} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
          </div>
          <div className="form-group">
            <label>Organizer</label>
            <input value={form.organizer ?? ''} onChange={e => setForm(f => ({ ...f, organizer: e.target.value }))} />
          </div>
          <div className="form-group">
            <label>Starts At</label>
            <input type="datetime-local" value={form.starts_at ?? ''} onChange={e => setForm(f => ({ ...f, starts_at: e.target.value }))} />
          </div>
          <div className="form-group">
            <label>Ends At</label>
            <input type="datetime-local" value={form.ends_at ?? ''} onChange={e => setForm(f => ({ ...f, ends_at: e.target.value }))} />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? 'Creating...' : 'Create'}
          </button>
        </form>
      )}

      <ErrorDisplay error={error} onRetry={fetchGames} />

      <DataTable<Game>
        columns={columns}
        data={games}
        loading={loading}
        emptyMessage="No games found"
        page={page}
        perPage={perPage}
        total={total}
        onPageChange={setPage}
        actions={(g) => (
          <ActionButton onClick={() => navigate(`/games/${g.id}`)}>View</ActionButton>
        )}
      />
    </div>
  )
}
