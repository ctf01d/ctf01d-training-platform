import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import * as gamesApi from '../api/games'
import type { Game, GameUpdate } from '../api/games'
import * as gameTeamsApi from '../api/game-teams'
import type { GameTeam, GameTeamCreate } from '../api/game-teams'
import * as servicesApi from '../api/services'
import type { Service } from '../api/services'
import * as resultsApi from '../api/results'
import type { Result, ResultCreate } from '../api/results'
import * as writeupsApi from '../api/writeups'
import type { Writeup, WriteupCreate } from '../api/writeups'
import * as teamsApi from '../api/teams'
import { ErrorDisplay, ActionButton, handleApiError } from '../components/ErrorDisplay'
import { useAuth } from '../auth/AuthContext'

export default function GameDetailPage() {
  const { id } = useParams<{ id: string }>()
  const gameId = Number(id)
  const navigate = useNavigate()
  const { user, isPlayer, isAdmin } = useAuth()

  const [game, setGame] = useState<Game | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<{ message?: string } | null>(null)

  const [editing, setEditing] = useState(false)
  const [editForm, setEditForm] = useState<GameUpdate>({})
  const [saving, setSaving] = useState(false)

  const [gameTeams, setGameTeams] = useState<GameTeam[]>([])
  const [teamNames, setTeamNames] = useState<Record<number, string>>({})
  const [manageableTeamIds, setManageableTeamIds] = useState<number[]>([])
  const [serviceIds, setServiceIds] = useState<number[]>([])
  const [serviceDetails, setServiceDetails] = useState<Record<number, Service>>({})
  const [results, setResults] = useState<Result[]>([])
  const [writeups, setWriteups] = useState<Writeup[]>([])

  const [addTeamForm, setAddTeamForm] = useState<GameTeamCreate>({ game_id: gameId, team_id: 0 })
  const [addServiceId, setAddServiceId] = useState('')
  const [addResultForm, setAddResultForm] = useState<ResultCreate>({ game_id: gameId, team_id: 0, score: 0 })
  const [addWriteupForm, setAddWriteupForm] = useState<WriteupCreate>({ game_id: gameId, team_id: 0, title: '', url: 'https://' })

  const fetchGame = useCallback(async () => {
    setLoading(true)
    const { data, error: err } = await gamesApi.getGame(gameId)
    if (err) setError(err)
    else if (data) setGame(data)
    setLoading(false)
  }, [gameId])

  const fetchGameTeams = useCallback(async () => {
    const { data } = await gamesApi.listGameTeams(gameId)
    if (data) {
      setGameTeams(data.items)
      const ids = data.items.map(gt => gt.team_id)
      const nameMap: Record<number, string> = {}
      const manageable: number[] = []
      for (const tid of ids) {
        const r = await teamsApi.getTeam(tid)
        if (r.data) nameMap[tid] = r.data.name
        if (isAdmin) {
          manageable.push(tid)
        } else if (user) {
          const r = await teamsApi.listTeamMembers(tid)
          const membership = r.data?.items.find(m => m.user_id === user.id)
          if (membership?.status === 'approved' && (membership.role === 'owner' || membership.role === 'captain' || membership.role === 'vice_captain')) {
            manageable.push(tid)
          }
        }
      }
      setTeamNames(prev => {
        let changed = false
        const next = { ...prev }
        for (const [tid, name] of Object.entries(nameMap)) {
          const teamId = Number(tid)
          if (next[teamId] !== name) {
            next[teamId] = name
            changed = true
          }
        }
        return changed ? next : prev
      })
      setManageableTeamIds(manageable)
    }
  }, [gameId, isAdmin, user])

  const fetchServices = useCallback(async () => {
    const { data } = await gamesApi.listGameServices(gameId)
    if (data) {
      setServiceIds(data)
      const details: Record<number, Service> = {}
      for (const sid of data) {
        const r = await servicesApi.getService(sid)
        if (r.data) details[sid] = r.data
      }
      setServiceDetails(details)
    }
  }, [gameId])

  const fetchResults = useCallback(async () => {
    const { data } = await resultsApi.listResults({ game_id: gameId })
    if (data) setResults(data.items)
  }, [gameId])

  const fetchWriteups = useCallback(async () => {
    const { data } = await writeupsApi.listWriteups({ game_id: gameId })
    if (data) setWriteups(data.items)
  }, [gameId])

  useEffect(() => {
    void fetchGame()
    void fetchGameTeams()
    void fetchServices()
    void fetchResults()
    void fetchWriteups()
  }, [fetchGame, fetchGameTeams, fetchServices, fetchResults, fetchWriteups])

  const handleSave = async () => {
    setSaving(true)
    const { data, error: err } = await gamesApi.updateGame(gameId, editForm)
    setSaving(false)
    if (err) { setError(err); return }
    if (data) { setGame(data); setEditing(false) }
  }

  const startEdit = () => {
    if (!game) return
    setEditForm({
      name: game.name ?? undefined,
      organizer: game.organizer ?? undefined,
      starts_at: game.starts_at ?? undefined,
      ends_at: game.ends_at ?? undefined,
      site_url: game.site_url ?? undefined,
      ctftime_url: game.ctftime_url ?? undefined,
      registration_opens_at: game.registration_opens_at ?? undefined,
      registration_closes_at: game.registration_closes_at ?? undefined,
      scoreboard_opens_at: game.scoreboard_opens_at ?? undefined,
      scoreboard_closes_at: game.scoreboard_closes_at ?? undefined,
      vpn_url: game.vpn_url ?? undefined,
      vpn_config_url: game.vpn_config_url ?? undefined,
      access_instructions: game.access_instructions ?? undefined,
      access_secret: game.access_secret ?? undefined,
    })
    setEditing(true)
  }

  const handleFinalize = async () => {
    const { data, error: err } = await gamesApi.finalizeGame(gameId)
    if (err) { setError(err); return }
    if (data) setGame(data)
  }

  const handleUnfinalize = async () => {
    const { data, error: err } = await gamesApi.unfinalizeGame(gameId)
    if (err) { setError(err); return }
    if (data) setGame(data)
  }

  const handleExportCtf01d = async () => {
    try {
      const { data, error: err } = await gamesApi.exportCtf01d(gameId)
      if (err) { setError(handleApiError(err)); return }
      if (data) {
        const blob = data as unknown as Blob
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `ctf01d-game-${gameId}.zip`
        a.click()
        URL.revokeObjectURL(url)
      }
    } catch (e) {
      setError(handleApiError(e))
    }
  }

  const handleAddTeam = async (e: React.FormEvent) => {
    e.preventDefault()
    const { error: err } = await gameTeamsApi.createGameTeam(addTeamForm)
    if (err) { setError(err); return }
    setAddTeamForm({ game_id: gameId, team_id: 0 })
    await fetchGameTeams()
  }

  const handleRemoveTeam = async (gtId: number) => {
    const { error: err } = await gameTeamsApi.deleteGameTeam(gtId)
    if (err) { setError(err); return }
    await fetchGameTeams()
  }

  const handleReorder = async () => {
    const items = gameTeams
      .sort((a, b) => a.order - b.order)
      .map((gt, i) => ({ id: gt.id, order: i + 1 }))
    const { error: err } = await gameTeamsApi.reorderGameTeams(gameId, items)
    if (err) { setError(err); return }
    await fetchGameTeams()
  }

  const handleAddService = async (e: React.FormEvent) => {
    e.preventDefault()
    const sid = Number(addServiceId)
    if (!sid) return
    const { error: err } = await gamesApi.addGameService(gameId, sid)
    if (err) { setError(err); return }
    setAddServiceId('')
    await fetchServices()
  }

  const handleRemoveService = async (sid: number) => {
    const { error: err } = await gamesApi.removeGameService(gameId, sid)
    if (err) { setError(err); return }
    await fetchServices()
  }

  const handleAddResult = async (e: React.FormEvent) => {
    e.preventDefault()
    const { error: err } = await resultsApi.createResult(addResultForm)
    if (err) { setError(err); return }
    setAddResultForm({ game_id: gameId, team_id: 0, score: 0 })
    await fetchResults()
  }

  const handleDeleteResult = async (rid: number) => {
    const { error: err } = await resultsApi.deleteResult(rid)
    if (err) { setError(err); return }
    await fetchResults()
  }

  const handleAddWriteup = async (e: React.FormEvent) => {
    e.preventDefault()
    const { error: err } = await writeupsApi.createWriteup(addWriteupForm)
    if (err) { setError(err); return }
    setAddWriteupForm({ game_id: gameId, team_id: 0, title: '', url: 'https://' })
    await fetchWriteups()
  }

  const handleDeleteWriteup = async (writeupId: number) => {
    const { error: err } = await writeupsApi.deleteWriteup(writeupId)
    if (err) { setError(err); return }
    await fetchWriteups()
  }

  if (loading) return <div className="loading">Loading...</div>
  if (!game) return <ErrorDisplay error={error} onRetry={fetchGame} />

  const canEdit = isPlayer
  const canManageWriteups = isAdmin || manageableTeamIds.length > 0

  return (
    <div className="page">
      <div className="page-header">
        <h1>{game.name ?? `Game #${game.id}`}</h1>
        <button className="btn btn-sm" onClick={() => navigate('/games')}>Back</button>
      </div>

      <ErrorDisplay error={error} onRetry={fetchGame} />

      {!editing ? (
        <div className="detail-card">
          <table className="detail-table">
            <tbody>
              <tr><td className="label">Name</td><td>{game.name ?? '—'}</td></tr>
              <tr><td className="label">Organizer</td><td>{game.organizer ?? '—'}</td></tr>
              <tr><td className="label">Status</td><td>{game.status ?? 'unknown'}</td></tr>
              <tr><td className="label">Starts At</td><td>{game.starts_at ? new Date(game.starts_at).toLocaleString() : '—'}</td></tr>
              <tr><td className="label">Ends At</td><td>{game.ends_at ? new Date(game.ends_at).toLocaleString() : '—'}</td></tr>
              <tr><td className="label">Registration Status</td><td>{game.registration_status ?? 'unscheduled'}</td></tr>
              <tr><td className="label">Registration Opens</td><td>{game.registration_opens_at ? new Date(game.registration_opens_at).toLocaleString() : '—'}</td></tr>
              <tr><td className="label">Registration Closes</td><td>{game.registration_closes_at ? new Date(game.registration_closes_at).toLocaleString() : '—'}</td></tr>
              <tr><td className="label">Scoreboard Status</td><td>{game.scoreboard_status ?? 'closed'}</td></tr>
              <tr><td className="label">Scoreboard Opens</td><td>{game.scoreboard_opens_at ? new Date(game.scoreboard_opens_at).toLocaleString() : '—'}</td></tr>
              <tr><td className="label">Scoreboard Closes</td><td>{game.scoreboard_closes_at ? new Date(game.scoreboard_closes_at).toLocaleString() : '—'}</td></tr>
              <tr><td className="label">Site URL</td><td>{game.site_url ? <a href={safeHref(game.site_url)} target="_blank" rel="noreferrer">{game.site_url}</a> : '—'}</td></tr>
              <tr><td className="label">CTFtime URL</td><td>{game.ctftime_url ? <a href={safeHref(game.ctftime_url)} target="_blank" rel="noreferrer">{game.ctftime_url}</a> : '—'}</td></tr>
              <tr><td className="label">VPN URL</td><td>{game.vpn_url ?? '—'}</td></tr>
              <tr><td className="label">Finalized</td><td>{game.finalized ? `Yes${game.finalized_at ? ` at ${new Date(game.finalized_at).toLocaleString()}` : ''}` : 'No'}</td></tr>
              {isAdmin && game.access_secret && <tr><td className="label">Access Secret</td><td>{game.access_secret}</td></tr>}
              {isAdmin && game.access_instructions && <tr><td className="label">Access Instructions</td><td>{game.access_instructions}</td></tr>}
            </tbody>
          </table>
          {canEdit && <button className="btn btn-sm" onClick={startEdit}>Edit</button>}
        </div>
      ) : (
        <form onSubmit={e => { e.preventDefault(); void handleSave() }} className="edit-form">
          {(['name', 'organizer', 'site_url', 'ctftime_url', 'vpn_url', 'vpn_config_url', 'access_instructions', 'access_secret'] as const).map(field => (
            <div className="form-group" key={field}>
              <label>{field.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())}</label>
              <input value={(editForm[field] as string) ?? ''} onChange={e => setEditForm(f => ({ ...f, [field]: e.target.value }))} />
            </div>
          ))}
          {(['starts_at', 'ends_at', 'registration_opens_at', 'registration_closes_at', 'scoreboard_opens_at', 'scoreboard_closes_at'] as const).map(field => (
            <div className="form-group" key={field}>
              <label>{field.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())}</label>
              <input type="datetime-local" value={(editForm[field] as string) ?? ''} onChange={e => setEditForm(f => ({ ...f, [field]: e.target.value }))} />
            </div>
          ))}
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
            <button type="button" className="btn" onClick={() => setEditing(false)}>Cancel</button>
          </div>
        </form>
      )}

      {canEdit && (
        <div className="detail-section">
          <h3>Actions</h3>
          <div className="action-buttons">
            {game.finalized ? (
              <ActionButton onClick={handleUnfinalize}>Unfinalize</ActionButton>
            ) : (
              <ActionButton onClick={handleFinalize} confirm="Finalize this game?">Finalize</ActionButton>
            )}
            <ActionButton onClick={handleExportCtf01d}>Export ctf01d</ActionButton>
            <ActionButton onClick={() => { void gamesApi.deleteGame(gameId).then(() => navigate('/games')) }} variant="danger" confirm="Delete this game?">Delete</ActionButton>
          </div>
        </div>
      )}

      <div className="detail-section">
        <h3>Roster (Game Teams)</h3>
        {gameTeams.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th>Order</th>
                <th>Team</th>
                <th>IP Address</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {gameTeams.sort((a, b) => a.order - b.order).map(gt => (
                <tr key={gt.id}>
                  <td>{gt.order}</td>
                  <td>{teamNames[gt.team_id] ?? `Team #${gt.team_id}`}</td>
                  <td>{gt.ip_address ?? '—'}</td>
                  <td>
                    {canEdit && (
                      <ActionButton onClick={() => void handleRemoveTeam(gt.id)} variant="danger">Remove</ActionButton>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p>No teams in roster.</p>
        )}
        {canEdit && (
          <>
            <form onSubmit={e => void handleAddTeam(e)} className="inline-form">
              <input type="number" placeholder="Team ID" value={addTeamForm.team_id || ''} onChange={e => setAddTeamForm(f => ({ ...f, team_id: Number(e.target.value) }))} required />
              <input placeholder="IP Address" value={addTeamForm.ip_address ?? ''} onChange={e => setAddTeamForm(f => ({ ...f, ip_address: e.target.value }))} />
              <button type="submit" className="btn btn-sm">Add Team</button>
            </form>
            <button className="btn btn-sm" onClick={() => void handleReorder()}>Reorder</button>
          </>
        )}
      </div>

      <div className="detail-section">
        <h3>Services</h3>
        {serviceIds.length > 0 ? (
          <ul>
            {serviceIds.map(sid => (
              <li key={sid}>
                <a href={`/services/${sid}`}>{serviceDetails[sid]?.name ?? `Service #${sid}`}</a>
                {canEdit && (
                  <ActionButton onClick={() => void handleRemoveService(sid)} variant="danger">Unlink</ActionButton>
                )}
              </li>
            ))}
          </ul>
        ) : (
          <p>No services linked.</p>
        )}
        {canEdit && (
          <form onSubmit={e => void handleAddService(e)} className="inline-form">
            <input type="number" placeholder="Service ID" value={addServiceId} onChange={e => setAddServiceId(e.target.value)} required />
            <button type="submit" className="btn btn-sm">Link Service</button>
          </form>
        )}
      </div>

      <div className="detail-section">
        <h3>Results</h3>
        {results.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr><th>ID</th><th>Team ID</th><th>Score</th><th>Actions</th></tr>
            </thead>
            <tbody>
              {results.map(r => (
                <tr key={r.id}>
                  <td>{r.id}</td>
                  <td>{r.team_id}</td>
                  <td>{r.score ?? '—'}</td>
                  <td>
                    {canEdit && (
                      <ActionButton onClick={() => void handleDeleteResult(r.id)} variant="danger" confirm="Delete this result?">Delete</ActionButton>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p>No results.</p>
        )}
        {canEdit && (
          <form onSubmit={e => void handleAddResult(e)} className="inline-form">
            <input type="number" placeholder="Team ID" value={addResultForm.team_id || ''} onChange={e => setAddResultForm(f => ({ ...f, team_id: Number(e.target.value) }))} required />
            <input type="number" placeholder="Score" value={addResultForm.score ?? ''} onChange={e => setAddResultForm(f => ({ ...f, score: Number(e.target.value) }))} required />
            <button type="submit" className="btn btn-sm">Add Result</button>
          </form>
        )}
      </div>

      <div className="detail-section">
        <h3>Writeups</h3>
        {writeups.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr><th>Team</th><th>Title</th><th>URL</th><th>Actions</th></tr>
            </thead>
            <tbody>
              {writeups.map(w => (
                <tr key={w.id}>
                  <td>{teamNames[w.team_id] ?? `Team #${w.team_id}`}</td>
                  <td>{w.title}</td>
                  <td><a href={safeHref(w.url)} target="_blank" rel="noreferrer">{w.url}</a></td>
                  <td>
                    {(isAdmin || manageableTeamIds.includes(w.team_id)) && (
                      <ActionButton onClick={() => void handleDeleteWriteup(w.id)} variant="danger" confirm="Delete this writeup?">Delete</ActionButton>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p>No writeups.</p>
        )}
        {canManageWriteups && (
          <form onSubmit={e => void handleAddWriteup(e)} className="inline-form">
            <input type="number" placeholder="Team ID" value={addWriteupForm.team_id || ''} onChange={e => setAddWriteupForm(f => ({ ...f, team_id: Number(e.target.value) }))} required />
            <input placeholder="Title" value={addWriteupForm.title} onChange={e => setAddWriteupForm(f => ({ ...f, title: e.target.value }))} required />
            <input placeholder="https://..." value={addWriteupForm.url} onChange={e => setAddWriteupForm(f => ({ ...f, url: e.target.value }))} required />
            <button type="submit" className="btn btn-sm">Add Writeup</button>
          </form>
        )}
      </div>
    </div>
  )
}

function safeHref(url: string): string {
  if (/^https?:\/\//i.test(url)) return url
  return 'about:blank'
}
