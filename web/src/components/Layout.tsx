import { Link, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

export default function Layout() {
  const { user, logout, isAdmin, isPlayer } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <div className="layout">
      <header className="header">
        <div className="header-inner">
          <Link to="/" className="logo">CTF01D Training Platform</Link>
          {user && (
            <nav className="nav">
              <Link to="/games">Games</Link>
              <Link to="/services">Services</Link>
              <Link to="/teams">Teams</Link>
              <Link to="/scoreboard">Scoreboard</Link>
              {isAdmin && (
                <>
                  <Link to="/universities">Universities</Link>
                  <Link to="/users">Users</Link>
                </>
              )}
              {isPlayer && <Link to="/results">Results</Link>}
            </nav>
          )}
          <div className="header-right">
            {user ? (
              <div className="user-menu">
                <Link to="/profile">{user.display_name}</Link>
                <span className="user-role">{user.role}</span>
                <button onClick={handleLogout} className="btn btn-sm">Logout</button>
              </div>
            ) : (
              <Link to="/login" className="btn btn-sm">Login</Link>
            )}
          </div>
        </div>
      </header>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
