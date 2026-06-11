import client from './client'
import type { components } from './schema'

export type Team = components['schemas']['Team']
export type TeamCreate = components['schemas']['TeamCreate']
export type TeamUpdate = components['schemas']['TeamUpdate']

export async function listTeams(query?: { page?: number; per_page?: number }) {
  return client.GET('/teams', { params: { query } })
}

export async function getTeam(id: number) {
  return client.GET('/teams/{id}', { params: { path: { id } } })
}

export async function createTeam(body: TeamCreate) {
  return client.POST('/teams', { body })
}

export async function updateTeam(id: number, body: TeamUpdate) {
  return client.PATCH('/teams/{id}', { params: { path: { id } }, body })
}

export async function deleteTeam(id: number) {
  return client.DELETE('/teams/{id}', { params: { path: { id } } })
}

export async function requestJoinTeam(id: number) {
  return client.POST('/teams/{id}/join-request', { params: { path: { id } } })
}

export async function inviteToTeam(id: number, userId: number) {
  return client.POST('/teams/{id}/invite', {
    params: { path: { id } },
    body: { user_id: userId },
  })
}

export async function listTeamMembers(id: number) {
  return client.GET('/teams/{id}/members', { params: { path: { id } } })
}

export async function listTeamEvents(id: number, query?: { page?: number; per_page?: number }) {
  return client.GET('/teams/{id}/events', { params: { path: { id }, query } })
}
