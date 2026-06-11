import client from './client'
import { setToken, clearToken } from './auth'
import type { components } from './schema'

export type User = components['schemas']['User']
export type LoginRequest = components['schemas']['LoginRequest']
export type LoginResponse = components['schemas']['LoginResponse']
export type UserCreate = components['schemas']['UserCreate']
export type UserUpdate = components['schemas']['UserUpdate']

export async function login(body: LoginRequest) {
  const { data, error } = await client.POST('/session', {
    body,
  })
  if (data) {
    setToken(data.token)
  }
  return { data, error }
}

export async function logout() {
  await client.DELETE('/session')
  clearToken()
}

export async function getProfile() {
  return client.GET('/profile')
}

export async function updateProfile(body: UserUpdate) {
  return client.PATCH('/profile', { body })
}

export async function listUsers(query?: { page?: number; per_page?: number }) {
  return client.GET('/users', { params: { query } })
}

export async function getUser(id: number) {
  return client.GET('/users/{id}', { params: { path: { id } } })
}

export async function createUser(body: UserCreate) {
  return client.POST('/users', { body })
}

export async function updateUser(id: number, body: UserUpdate) {
  return client.PATCH('/users/{id}', { params: { path: { id } }, body })
}

export async function deleteUser(id: number) {
  return client.DELETE('/users/{id}', { params: { path: { id } } })
}
