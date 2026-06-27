export type ThemeId = "classic" | "indigo" | "dark" | "midnight";
export type ThemeKind = "light" | "dark";

export interface ThemeOption {
  id: ThemeId;
  label: string;
  kind: ThemeKind;
}

export const THEMES: ThemeOption[] = [
  { id: "classic", label: "Classic", kind: "light" },
  { id: "indigo", label: "Indigo", kind: "light" },
  { id: "dark", label: "Dark", kind: "dark" },
  { id: "midnight", label: "Midnight", kind: "dark" },
];

export const DEFAULT_THEME: ThemeId = "classic";
const STORAGE_KEY = "ctf01d-theme";

export function getStoredTheme(): ThemeId {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    if (value && THEMES.some((t) => t.id === value)) {
      return value as ThemeId;
    }
  } catch {
    /* localStorage unavailable (private mode / SSR) */
  }
  return DEFAULT_THEME;
}

export function applyTheme(theme: ThemeId): void {
  const root = document.documentElement;
  if (theme === DEFAULT_THEME) {
    root.removeAttribute("data-theme");
  } else {
    root.setAttribute("data-theme", theme);
  }
}

export function setTheme(theme: ThemeId): void {
  applyTheme(theme);
  try {
    localStorage.setItem(STORAGE_KEY, theme);
  } catch {
    /* ignore persistence failures */
  }
}
