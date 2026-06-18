import type { Middleware } from "openapi-fetch";

const TOKEN_KEY = "auth_token";

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

const authMiddleware: Middleware = {
  async onRequest({ request }) {
    const token = getToken();
    if (token) {
      request.headers.set("Authorization", `Bearer ${token}`);
    }
    return request;
  },
};

const unauthorizedMiddleware: Middleware = {
  async onResponse({ response }) {
    // Only force a redirect when an existing session became invalid.
    // Guests (no token) may freely browse public pages where some
    // requests legitimately return 401, so they must not be bounced.
    if (
      response.status === 401 &&
      getToken() &&
      !window.location.pathname.startsWith("/login")
    ) {
      clearToken();
      window.location.href = "/login";
    }
    return response;
  },
};

export { authMiddleware, unauthorizedMiddleware };
