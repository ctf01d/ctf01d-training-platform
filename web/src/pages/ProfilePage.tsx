import { useState, useCallback } from 'react'
import * as usersApi from '../api/users'
import type { UserUpdate } from '../api/users'
import { ErrorDisplay } from '../components/ErrorDisplay'
import { useAuth } from '../auth/AuthContext'

export default function ProfilePage() {
  const { user, refreshUser } = useAuth()

  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState<UserUpdate>({})
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<{ message?: string } | null>(null)
  const [success, setSuccess] = useState(false)

  const startEdit = useCallback(() => {
    if (!user) return
    setForm({
      display_name: user.display_name,
      avatar_url: user.avatar_url ?? undefined,
    })
    setEditing(true)
    setSuccess(false)
  }, [user])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError(null)
    const { data, error: err } = await usersApi.updateProfile(form)
    setSaving(false)
    if (err) { setError(err); return }
    if (data) {
      await refreshUser()
      setEditing(false)
      setSuccess(true)
    }
  }

  if (!user) return <div className="loading">Loading...</div>

  return (
    <div className="page">
      <div className="page-header">
        <h1>Profile</h1>
        {!editing && (
          <button className="btn btn-sm" onClick={startEdit}>Edit</button>
        )}
      </div>

      <ErrorDisplay error={error} />
      {success && <div className="success-message">Profile updated successfully.</div>}

      {!editing ? (
        <div className="detail-card">
          <table className="detail-table">
            <tbody>
              <tr><td className="label">ID</td><td>{user.id}</td></tr>
              <tr><td className="label">Username</td><td>{user.user_name}</td></tr>
              <tr><td className="label">Display Name</td><td>{user.display_name}</td></tr>
              <tr><td className="label">Role</td><td>{user.role}</td></tr>
              <tr><td className="label">Rating</td><td>{user.rating}</td></tr>
              <tr>
                <td className="label">Avatar</td>
                <td>
                  {user.avatar_url ? (
                    <img src={user.avatar_url} alt="Avatar" className="profile-avatar" />
                  ) : '—'}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      ) : (
        <form onSubmit={handleSave} className="edit-form">
          <div className="form-group">
            <label>Display Name</label>
            <input value={form.display_name ?? ''} onChange={e => setForm(f => ({ ...f, display_name: e.target.value }))} />
          </div>
          <div className="form-group">
            <label>Avatar URL</label>
            <input value={form.avatar_url ?? ''} onChange={e => setForm(f => ({ ...f, avatar_url: e.target.value || null }))} />
          </div>
          <div className="form-group">
            <label>New Password</label>
            <input type="password" placeholder="Leave blank to keep current" value={form.password ?? ''} onChange={e => setForm(f => ({ ...f, password: e.target.value || undefined }))} />
          </div>
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
            <button type="button" className="btn" onClick={() => { setEditing(false); setSuccess(false) }}>Cancel</button>
          </div>
        </form>
      )}
    </div>
  )
}
