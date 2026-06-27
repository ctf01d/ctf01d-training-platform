import { useEffect } from "react";
import { useAuth } from "../auth/AuthContext";
import { setTheme } from "../theme";

// Applies the logged-in user's saved theme (and caches it locally) whenever the
// authenticated user — or their stored theme — changes. Pre-auth, main.tsx has
// already applied the locally cached theme, so there is no flash on reload.
export default function ThemeSync() {
  const { user } = useAuth();

  useEffect(() => {
    if (user?.theme) setTheme(user.theme);
  }, [user?.theme]);

  return null;
}
