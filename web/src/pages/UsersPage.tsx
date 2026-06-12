import { useState, useEffect, useCallback } from 'react'
import * as usersApi from '../api/users'
import type { User, UserCreate } from '../api/users'
import { CardGrid, EntityCard, CardBadge, CardMeta, Pagination } from '../components/Card'
import { ErrorDisplay, ActionButton } from '../components/ErrorDisplay'

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<{ message?: string } | null>(null)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const perPage = 20

  const [showCreate, setShowCreate] = useState(false)
  const [createForm, setCreateForm] = useState<UserCreate>({ user_name: '', display_name: '', password: '', role: 'guest' })
  const [creating, setCreating] = useState(false)

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    setError(null)
    const { data, error: err } = await usersApi.listUsers({ page, per_page: perPage })
    if (err) {
      setError(err)
    } else if (data) {
      setUsers(data.items)
      setTotal(data.pagination.total)
    }
    setLoading(false)
  }, [page])

  useEffect(() => {
    void fetchUsers()
  }, [fetchUsers])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)
    const { error: err } = await usersApi.createUser(createForm)
    setCreating(false)
    if (err) { setError(err); return }
    setCreateForm({ user_name: '', display_name: '', password: '', role: 'guest' })
    setShowCreate(false)
    await fetchUsers()
  }

  const handleDelete = async (id: number) => {
    const { error: err } = await usersApi.deleteUser(id)
    if (err) { setError(err); return }
    await fetchUsers()
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1>Users</h1>
        <button className="btn btn-primary" onClick={() => setShowCreate(!showCreate)}>
          {showCreate ? 'Cancel' : 'Create User'}
        </button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="create-form">
          <div className="form-group">
            <label>Username</label>
            <input value={createForm.user_name} onChange={e => setCreateForm(f => ({ ...f, user_name: e.target.value }))} required />
          </div>
          <div className="form-group">
            <label>Display Name</label>
            <input value={createForm.display_name} onChange={e => setCreateForm(f => ({ ...f, display_name: e.target.value }))} required />
          </div>
          <div className="form-group">
            <label>Password</label>
            <input type="password" value={createForm.password} onChange={e => setCreateForm(f => ({ ...f, password: e.target.value }))} required />
          </div>
          <div className="form-group">
            <label>Role</label>
            <select value={createForm.role} onChange={e => setCreateForm(f => ({ ...f, role: e.target.value as UserCreate['role'] }))}>
              <option value="guest">Guest</option>
              <option value="player">Player</option>
              <option value="admin">Admin</option>
            </select>
          </div>
          <div className="form-group">
            <label>Avatar URL</label>
            <input value={createForm.avatar_url ?? ''} onChange={e => setCreateForm(f => ({ ...f, avatar_url: e.target.value || null }))} />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? 'Creating...' : 'Create'}
          </button>
        </form>
      )}

      <ErrorDisplay error={error} onRetry={fetchUsers} />

      <CardGrid loading={loading} isEmpty={users.length === 0} emptyMessage="No users found">
        {users.map((u) => (
          <EntityCard
            key={u.id}
            avatarUrl={u.avatar_url}
            avatarText={u.display_name || u.user_name}
            title={u.display_name}
            badges={<CardBadge variant={u.role}>{u.role}</CardBadge>}
            footer={
              <ActionButton onClick={() => handleDelete(u.id)} variant="danger" confirm="Delete this user?">
                Delete
              </ActionButton>
            }
          >
            <CardMeta label="Username">@{u.user_name}</CardMeta>
            <CardMeta label="Rating">{u.rating}</CardMeta>
          </EntityCard>
        ))}
      </CardGrid>

      <Pagination page={page} perPage={perPage} total={total} onPageChange={setPage} />
    </div>
  )
}
