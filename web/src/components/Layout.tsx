import { useEffect, useRef, useState } from "react";
import {
  Link,
  NavLink,
  Outlet,
  useLocation,
  useNavigate,
} from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

const primaryLinks = [
  { to: "/games", label: "Games" },
  { to: "/teams", label: "Teams" },
  { to: "/scoreboard", label: "Scoreboard" },
];

export default function Layout() {
  const { user, logout, isAdmin, isPlayer } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [openMenu, setOpenMenu] = useState<"admin" | "account" | null>(null);
  const adminMenuRef = useRef<HTMLDivElement>(null);
  const accountMenuRef = useRef<HTMLDivElement>(null);
  const adminActive = ["/services", "/universities", "/users"].some((path) =>
    location.pathname.startsWith(path),
  );
  const accountPaths = ["/profile"];
  const accountActive = accountPaths.some((path) =>
    location.pathname.startsWith(path),
  );
  const adminOpen = openMenu === "admin";
  const accountOpen = openMenu === "account";

  useEffect(() => {
    setOpenMenu(null);
  }, [location.pathname]);

  useEffect(() => {
    if (!openMenu) return;

    const handlePointerDown = (event: MouseEvent | TouchEvent) => {
      const target = event.target;
      if (!(target instanceof Node)) return;
      if (adminMenuRef.current?.contains(target)) return;
      if (accountMenuRef.current?.contains(target)) return;
      setOpenMenu(null);
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpenMenu(null);
    };

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("touchstart", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("touchstart", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [openMenu]);

  const toggleMenu = (menu: "admin" | "account") => {
    setOpenMenu((current) => (current === menu ? null : menu));
  };

  const closeMenu = () => {
    setOpenMenu(null);
  };

  const handleLogout = async () => {
    closeMenu();
    await logout();
    navigate("/login");
  };
  const userInitials = (
    (user?.display_name ?? user?.user_name ?? "?").trim().slice(0, 2) || "?"
  ).toUpperCase();
  const userAvatar = user?.avatar_url?.trim();

  return (
    <div className="layout">
      <header className="header">
        <div className="header-inner">
          <Link to="/" className="logo" aria-label="CTF01D Training Platform">
            <span className="logo-mark">01</span>
            <span className="logo-text">CTF01D</span>
          </Link>
          <nav className="nav" aria-label="Primary navigation">
            {primaryLinks.map((link) => (
              <NavLink key={link.to} to={link.to}>
                {link.label}
              </NavLink>
            ))}
            {isPlayer && <NavLink to="/results">Results</NavLink>}
            {isAdmin && (
              <div
                className={`nav-menu ${adminActive ? "is-active" : ""} ${
                  adminOpen ? "is-open" : ""
                }`}
                ref={adminMenuRef}
              >
                <button
                  type="button"
                  className="nav-menu-trigger"
                  aria-haspopup="menu"
                  aria-expanded={adminOpen}
                  onClick={() => toggleMenu("admin")}
                >
                  Admin
                </button>
                <div className="nav-menu-content" role="menu">
                  <NavLink to="/services" role="menuitem" onClick={closeMenu}>
                    Services
                  </NavLink>
                  <NavLink
                    to="/universities"
                    role="menuitem"
                    onClick={closeMenu}
                  >
                    Universities
                  </NavLink>
                  <NavLink to="/users" role="menuitem" onClick={closeMenu}>
                    Users
                  </NavLink>
                </div>
              </div>
            )}
            {!isAdmin && <NavLink to="/services">Services</NavLink>}
          </nav>
          <div className="header-right">
            {user ? (
              <div
                className={`account-menu ${accountActive ? "is-active" : ""} ${
                  accountOpen ? "is-open" : ""
                }`}
                ref={accountMenuRef}
              >
                <button
                  type="button"
                  className="account-menu-trigger"
                  aria-haspopup="menu"
                  aria-expanded={accountOpen}
                  aria-label="Account menu"
                  onClick={() => toggleMenu("account")}
                >
                  <span className="account-avatar">
                    {userAvatar ? (
                      <img src={userAvatar} alt="" />
                    ) : (
                      <span>{userInitials}</span>
                    )}
                  </span>
                </button>
                <div className="account-menu-content" role="menu">
                  <div className="account-menu-meta">
                    <span className="account-avatar account-avatar-lg">
                      {userAvatar ? (
                        <img src={userAvatar} alt="" />
                      ) : (
                        <span>{userInitials}</span>
                      )}
                    </span>
                    <span className="account-menu-identity">
                      <span className="account-menu-name">
                        {user.display_name}
                      </span>
                      <span className="user-role">{user.role}</span>
                    </span>
                  </div>
                  <NavLink to="/profile" role="menuitem" onClick={closeMenu}>
                    Profile
                  </NavLink>
                  <button
                    type="button"
                    className="account-menu-action"
                    onClick={handleLogout}
                    role="menuitem"
                  >
                    Logout
                  </button>
                </div>
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
