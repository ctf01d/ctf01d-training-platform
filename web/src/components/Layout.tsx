import { useEffect, useRef, useState } from "react";
import {
  Link,
  NavLink,
  Outlet,
  useLocation,
  useNavigate,
} from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useI18n } from "../i18n/I18nContext";
import { THEMES, applyTheme, getStoredTheme, setTheme, type ThemeId } from "../theme";

export default function Layout() {
  const { user, logout, isAdmin, isPlayer } = useAuth();
  const { t, roleLabel } = useI18n();
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
  const [theme, setThemeState] = useState<ThemeId>(getStoredTheme);

  const chooseTheme = (id: ThemeId) => {
    setTheme(id);
    setThemeState(id);
  };

  useEffect(() => {
    // Keep the DOM in sync with the stored choice (e.g. after a fresh mount).
    applyTheme(theme);
  }, [theme]);

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
  const primaryLinks = [
    { to: "/games", label: t("Games") },
    { to: "/teams", label: t("Teams") },
    { to: "/scoreboard", label: t("Scoreboard") },
  ];

  return (
    <div className="layout">
      <header className="header">
        <div className="header-inner">
          <Link to="/" className="logo" aria-label="CTF01D Training Platform">
            <span className="logo-mark">01</span>
            <span className="logo-text">CTF01D</span>
          </Link>
          <nav className="nav" aria-label={t("Primary navigation")}>
            {primaryLinks.map((link) => (
              <NavLink key={link.to} to={link.to}>
                {link.label}
              </NavLink>
            ))}
            {isPlayer && <NavLink to="/results">{t("Results")}</NavLink>}
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
                  {t("Admin")}
                </button>
                <div className="nav-menu-content" role="menu">
                  <NavLink to="/services" role="menuitem" onClick={closeMenu}>
                    {t("Services")}
                  </NavLink>
                  <NavLink
                    to="/universities"
                    role="menuitem"
                    onClick={closeMenu}
                  >
                    {t("Universities")}
                  </NavLink>
                  <NavLink to="/users" role="menuitem" onClick={closeMenu}>
                    {t("Users")}
                  </NavLink>
                </div>
              </div>
            )}
            {!isAdmin && <NavLink to="/services">{t("Services")}</NavLink>}
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
                  aria-label={t("Account menu")}
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
                      <span className="user-role">{roleLabel(user.role)}</span>
                    </span>
                  </div>
                  <div className="account-menu-themes" role="group" aria-label="Theme">
                    <span className="account-menu-section">Theme</span>
                    <div className="theme-options">
                      {THEMES.map((option) => (
                        <button
                          key={option.id}
                          type="button"
                          className={`theme-swatch theme-swatch--${option.id} ${
                            theme === option.id ? "is-active" : ""
                          }`}
                          onClick={() => chooseTheme(option.id)}
                          aria-pressed={theme === option.id}
                          title={option.label}
                        >
                          <span className="theme-swatch-dot" aria-hidden="true" />
                          <span className="theme-swatch-label">{option.label}</span>
                        </button>
                      ))}
                    </div>
                  </div>
                  <NavLink to="/profile" role="menuitem" onClick={closeMenu}>
                    {t("Profile")}
                  </NavLink>
                  <button
                    type="button"
                    className="account-menu-action"
                    onClick={handleLogout}
                    role="menuitem"
                  >
                    {t("Logout")}
                  </button>
                </div>
              </div>
            ) : (
              <Link to="/login" className="btn btn-sm">
                {t("Login")}
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
