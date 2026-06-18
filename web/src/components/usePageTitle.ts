import { useEffect } from "react";

const BASE_TITLE = "CTF01D Training Platform";

/**
 * Sets document.title to "<title> · CTF01D Training Platform" while the page is
 * mounted, restoring the base title on unmount. Pass undefined/empty (e.g. while
 * a detail entity is still loading) to show just the base title.
 */
export function usePageTitle(title?: string | null) {
  useEffect(() => {
    document.title = title ? `${title} · ${BASE_TITLE}` : BASE_TITLE;
    return () => {
      document.title = BASE_TITLE;
    };
  }, [title]);
}
