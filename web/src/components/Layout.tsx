import { Link, NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

const primaryLinks = [
  { to: "/games", label: "Games" },
  { to: "/services", label: "Services" },
  { to: "/teams", label: "Teams" },
  { to: "/scoreboard", label: "Scoreboard" },
];

export default function Layout() {
  const { user, logout, isAdmin, isPlayer } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate("/login");
  };

  return (
    <div className="layout">
      <header className="header">
        <div className="header-inner">
          <Link to="/" className="logo" aria-label="CTF01D Training Platform">
            <span className="logo-mark">01</span>
            <span className="logo-text">CTF01D</span>
          </Link>
          {user && (
            <nav className="nav" aria-label="Primary navigation">
              {primaryLinks.map((link) => (
                <NavLink key={link.to} to={link.to}>
                  {link.label}
                </NavLink>
              ))}
              {isAdmin && (
                <>
                  <NavLink to="/universities">Universities</NavLink>
                  <NavLink to="/users">Users</NavLink>
                </>
              )}
              {isPlayer && <NavLink to="/results">Results</NavLink>}
            </nav>
          )}
          <div className="header-right">
            {user ? (
              <div className="user-menu">
                <Link to="/profile" className="user-link">
                  {user.display_name}
                </Link>
                <span className="user-role">{user.role}</span>
                <button onClick={handleLogout} className="btn btn-sm">
                  Logout
                </button>
              </div>
            ) : (
              <Link to="/login" className="btn btn-sm">
                Login
              </Link>
            )}
          </div>
        </div>
      </header>
      <main className="main">
        <Outlet />
      </main>
    </div>
  );
}
