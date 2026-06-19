import client from "./client";
import { clearToken, getToken } from "./auth";
import type { components } from "./schema";

export type User = components["schemas"]["User"];
export type LoginRequest = components["schemas"]["LoginRequest"];
export type LoginResponse = components["schemas"]["LoginResponse"];
export type UserCreate = components["schemas"]["UserCreate"];
export type UserUpdate = components["schemas"]["UserUpdate"];
export type UserProfileUpdate = components["schemas"]["UserProfileUpdate"];
export type UserSession = components["schemas"]["UserSession"];
export type UserRole = User["role"];

export async function login(body: LoginRequest) {
  const { data, error } = await client.POST("/session", {
    body,
  });
  return { data, error };
}

export async function logout() {
  await client.DELETE("/session");
  clearToken();
}

export async function getProfile() {
  return client.GET("/profile");
}

export async function updateProfile(body: UserProfileUpdate) {
  return client.PATCH("/profile", { body });
}

export async function listProfileSessions() {
  return client.GET("/profile/sessions");
}

export async function listUsers(query?: {
  page?: number;
  per_page?: number;
  q?: string;
}) {
  return client.GET("/users", { params: { query } });
}

export async function getUser(id: number) {
  return client.GET("/users/{id}", { params: { path: { id } } });
}

export async function createUser(body: UserCreate) {
  return client.POST("/users", { body });
}

export async function updateUser(id: number, body: UserUpdate) {
  return client.PATCH("/users/{id}", { params: { path: { id } }, body });
}

export async function deleteUser(id: number, password: string) {
  return client.DELETE("/users/{id}", {
    params: { path: { id } },
    body: { password },
  });
}

export async function updateUserProfileAdmin(
  id: number,
  body: UserProfileUpdate,
) {
  return client.PATCH("/users/{id}/profile", {
    params: { path: { id } },
    body,
  });
}

export async function updateUserRole(id: number, role: UserRole) {
  return client.PATCH("/users/{id}/role", {
    params: { path: { id } },
    body: { role },
  });
}

export async function setUserBlocked(id: number, blocked: boolean) {
  return client.POST("/users/{id}/block", {
    params: { path: { id } },
    body: { blocked },
  });
}

export async function listUserSessions(id: number) {
  return client.GET("/users/{id}/sessions", { params: { path: { id } } });
}

export async function revokeUserSession(id: number, sessionId: number) {
  return client.DELETE("/users/{id}/sessions/{sessionId}", {
    params: { path: { id, sessionId } },
  });
}

// Avatar upload uses multipart/form-data which openapi-fetch does not model, so
// post the file directly with the auth header.
export async function uploadUserAvatar(id: number, file: File) {
  const formData = new FormData();
  formData.append("avatar", file);
  const token = getToken();
  const response = await fetch(`/api/v1/users/${id}/avatar`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: formData,
  });
  return response;
}

export async function uploadProfileAvatar(file: File) {
  const formData = new FormData();
  formData.append("avatar", file);
  const token = getToken();
  const response = await fetch("/api/v1/profile/avatar", {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: formData,
  });
  return response;
}
