import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import * as teamsApi from '../api/teams'
import type { Team, TeamUpdate } from '../api/teams'
import * as membershipsApi from '../api/team-memberships'
import type { TeamMembership, SetRoleRequest } from '../api/team-memberships'
import type { components } from '../api/schema'
import { ErrorDisplay, ActionButton } from '../components/ErrorDisplay'
import { useAuth } from '../auth/AuthContext'

type TeamMembershipEvent = components['schemas']['TeamMembershipEvent']

export default function TeamDetailPage() {
  const { id } = useParams<{ id: string }>()
  const teamId = Number(id)
  const navigate = useNavigate()
  const { user, isAdmin } = useAuth()

  const [team, setTeam] = useState<Team | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<{ message?: string } | null>(null)

  const [editing, setEditing] = useState(false)
  const [editForm, setEditForm] = useState<TeamUpdate>({})
  const [saving, setSaving] = useState(false)

  const [members, setMembers] = useState<TeamMembership[]>([])
  const [joinLoading, setJoinLoading] = useState(false)
  const [inviteUserId, setInviteUserId] = useState('')
  const [inviteLoading, setInviteLoading] = useState(false)

  const [events, setEvents] = useState<TeamMembershipEvent[]>([])
  const [eventsPage, setEventsPage] = useState(1)
  const [eventsTotal, setEventsTotal] = useState(0)
  const eventsPerPage = 10

  const [roleForm, setRoleForm] = useState<Record<number, string>>({})

  const fetchTeam = useCallback(async () => {
    setLoading(true)
    const { data, error: err } = await teamsApi.getTeam(teamId)
    if (err) setError(err)
    else if (data) setTeam(data)
    setLoading(false)
  }, [teamId])

  const fetchMembers = useCallback(async () => {
    const { data } = await teamsApi.listTeamMembers(teamId)
    if (data) setMembers(data.items)
  }, [teamId])

  const fetchEvents = useCallback(async () => {
    const { data } = await teamsApi.listTeamEvents(teamId, { page: eventsPage, per_page: eventsPerPage })
    if (data) {
      setEvents(data.items)
      setEventsTotal(data.pagination.total)
    }
  }, [teamId, eventsPage])

  useEffect(() => {
    void fetchTeam()
    void fetchMembers()
  }, [fetchTeam, fetchMembers])

  useEffect(() => {
    void fetchEvents()
  }, [fetchEvents])

  const isManager = isAdmin || members.some(
    m => m.user_id === user?.id && (m.role === 'owner' || m.role === 'captain') && m.status === 'approved'
  )

  const startEdit = () => {
    if (!team) return
    setEditForm({
      name: team.name,
      description: team.description ?? undefined,
      website: team.website ?? undefined,
      avatar_url: team.avatar_url ?? undefined,
      university_id: team.university_id ?? undefined,
    })
    setEditing(true)
  }

  const handleSave = async () => {
    setSaving(true)
    const { data, error: err } = await teamsApi.updateTeam(teamId, editForm)
    setSaving(false)
    if (err) { setError(err); return }
    if (data) { setTeam(data); setEditing(false) }
  }

  const handleJoin = async () => {
    setJoinLoading(true)
    const { error: err } = await teamsApi.requestJoinTeam(teamId)
    setJoinLoading(false)
    if (err) { setError(err); return }
    await fetchMembers()
  }

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault()
    const uid = Number(inviteUserId)
    if (!uid) return
    setInviteLoading(true)
    const { error: err } = await teamsApi.inviteToTeam(teamId, uid)
    setInviteLoading(false)
    if (err) { setError(err); return }
    setInviteUserId('')
    await fetchMembers()
  }

  const handleMembershipAction = async (action: () => Promise<{ data?: unknown; error?: { message: string } }>) => {
    const { error: err } = await action()
    if (err) { setError(err); return }
    await fetchMembers()
  }

  const handleSetRole = async (membershipId: number, role: string) => {
    const { error: err } = await membershipsApi.setTeamMembershipRole(membershipId, { role: role as SetRoleRequest['role'] })
    if (err) { setError(err); return }
    await fetchMembers()
  }

  const handleDelete = async () => {
    const { error: err } = await teamsApi.deleteTeam(teamId)
    if (err) { setError(err); return }
    navigate('/teams')
  }

  if (loading) return <div className="loading">Loading...</div>
  if (!team) return <ErrorDisplay error={error} onRetry={fetchTeam} />

  return (
    <div className="page">
      <div className="page-header">
        <h1>{team.name}</h1>
        <button className="btn btn-sm" onClick={() => navigate('/teams')}>Back</button>
      </div>

      <ErrorDisplay error={error} />

      {!editing ? (
        <div className="detail-card">
          <table className="detail-table">
            <tbody>
              <tr><td className="label">Name</td><td>{team.name}</td></tr>
              <tr><td className="label">Description</td><td>{team.description ?? '—'}</td></tr>
              <tr><td className="label">Website</td><td>{team.website ? <a href={safeUrl(team.website)} target="_blank" rel="noreferrer">{team.website}</a> : '—'}</td></tr>
              <tr><td className="label">Avatar</td><td>{team.avatar_url ? <a href={safeUrl(team.avatar_url)} target="_blank" rel="noreferrer">Link</a> : '—'}</td></tr>
              <tr><td className="label">Captain ID</td><td>{team.captain_id ?? '—'}</td></tr>
              <tr><td className="label">University ID</td><td>{team.university_id ?? '—'}</td></tr>
            </tbody>
          </table>
          <div className="action-buttons">
            {isManager && <button className="btn btn-sm" onClick={startEdit}>Edit</button>}
            {user && !isManager && (
              <button className="btn btn-primary" onClick={() => void handleJoin()} disabled={joinLoading}>
                {joinLoading ? 'Requesting...' : 'Request to Join'}
              </button>
            )}
            {isManager && (
              <ActionButton onClick={handleDelete} variant="danger" confirm="Delete this team?">Delete Team</ActionButton>
            )}
          </div>
        </div>
      ) : (
        <form onSubmit={e => { e.preventDefault(); void handleSave() }} className="edit-form">
          <div className="form-group">
            <label>Name</label>
            <input value={editForm.name ?? ''} onChange={e => setEditForm(f => ({ ...f, name: e.target.value }))} />
          </div>
          <div className="form-group">
            <label>Description</label>
            <input value={editForm.description ?? ''} onChange={e => setEditForm(f => ({ ...f, description: e.target.value }))} />
          </div>
          <div className="form-group">
            <label>Website</label>
            <input value={editForm.website ?? ''} onChange={e => setEditForm(f => ({ ...f, website: e.target.value }))} />
          </div>
          <div className="form-group">
            <label>Avatar URL</label>
            <input value={editForm.avatar_url ?? ''} onChange={e => setEditForm(f => ({ ...f, avatar_url: e.target.value }))} />
          </div>
          <div className="form-group">
            <label>University ID</label>
            <input type="number" value={editForm.university_id ?? ''} onChange={e => setEditForm(f => ({ ...f, university_id: e.target.value ? Number(e.target.value) : null }))} />
          </div>
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
            <button type="button" className="btn" onClick={() => setEditing(false)}>Cancel</button>
          </div>
        </form>
      )}

      <div className="detail-section">
        <h3>Members</h3>
        {members.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>
                <th>User ID</th>
                <th>Role</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {members.map(m => (
                <tr key={m.id}>
                  <td>{m.user_id}</td>
                  <td>
                    {isManager ? (
                      <select
                        value={roleForm[m.id] ?? m.role}
                        onChange={e => setRoleForm(prev => ({ ...prev, [m.id]: e.target.value }))}
                        onBlur={() => {
                          const newRole = roleForm[m.id]
                          if (newRole && newRole !== m.role) void handleSetRole(m.id, newRole)
                        }}
                      >
                        {(['owner', 'captain', 'vice_captain', 'player', 'guest'] as const).map(r => (
                          <option key={r} value={r}>{r}</option>
                        ))}
                      </select>
                    ) : m.role}
                  </td>
                  <td>
                    <span style={{
                      backgroundColor: m.status === 'approved' ? '#22c55e' : m.status === 'pending' ? '#f59e0b' : '#ef4444',
                      color: '#fff', padding: '2px 8px', borderRadius: 999, fontSize: 12,
                    }}>
                      {m.status}
                    </span>
                  </td>
                  <td>
                    {isManager && m.status === 'pending' && (
                      <>
                        <ActionButton onClick={() => void handleMembershipAction(() => membershipsApi.approveTeamMembership(m.id))} variant="success">Approve</ActionButton>
                        <ActionButton onClick={() => void handleMembershipAction(() => membershipsApi.rejectTeamMembership(m.id))} variant="danger">Reject</ActionButton>
                      </>
                    )}
                    {m.user_id === user?.id && m.status === 'pending' && (
                      <>
                        <ActionButton onClick={() => void handleMembershipAction(() => membershipsApi.acceptTeamMembership(m.id))} variant="success">Accept</ActionButton>
                        <ActionButton onClick={() => void handleMembershipAction(() => membershipsApi.declineTeamMembership(m.id))} variant="danger">Decline</ActionButton>
                      </>
                    )}
                    {isManager && (
                      <ActionButton onClick={() => void handleMembershipAction(() => membershipsApi.deleteTeamMembership(m.id))} variant="danger" confirm="Remove this member?">Remove</ActionButton>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p>No members.</p>
        )}

        {isManager && (
          <form onSubmit={e => void handleInvite(e)} className="inline-form" style={{ marginTop: 12 }}>
            <input
              type="number"
              placeholder="User ID to invite"
              value={inviteUserId}
              onChange={e => setInviteUserId(e.target.value)}
              required
            />
            <button type="submit" className="btn btn-sm" disabled={inviteLoading}>
              {inviteLoading ? 'Inviting...' : 'Invite'}
            </button>
          </form>
        )}
      </div>

      <div className="detail-section">
        <h3>Events</h3>
        {events.length > 0 ? (
          <>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>User ID</th>
                  <th>Action</th>
                  <th>Role Change</th>
                  <th>Status Change</th>
                </tr>
              </thead>
              <tbody>
                {events.map(ev => (
                  <tr key={ev.id}>
                    <td>{ev.created_at ? new Date(ev.created_at).toLocaleString() : '—'}</td>
                    <td>{ev.user_id}</td>
                    <td>{ev.action}</td>
                    <td>{ev.from_role && ev.to_role ? `${ev.from_role} → ${ev.to_role}` : '—'}</td>
                    <td>{ev.from_status && ev.to_status ? `${ev.from_status} → ${ev.to_status}` : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {Math.ceil(eventsTotal / eventsPerPage) > 1 && (
              <div className="pagination">
                <button disabled={eventsPage <= 1} onClick={() => setEventsPage(eventsPage - 1)}>Prev</button>
                <span>Page {eventsPage} of {Math.ceil(eventsTotal / eventsPerPage)}</span>
                <button disabled={eventsPage >= Math.ceil(eventsTotal / eventsPerPage)} onClick={() => setEventsPage(eventsPage + 1)}>Next</button>
              </div>
            )}
          </>
        ) : (
          <p>No events.</p>
        )}
      </div>
    </div>
  )
}

function safeUrl(url: string): string {
  if (/^https?:\/\//i.test(url)) return url
  return 'about:blank'
}
