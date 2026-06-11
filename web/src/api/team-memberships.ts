import client from './client'
import type { components } from './schema'

export type TeamMembership = components['schemas']['TeamMembership']
export type TeamMembershipCreate = components['schemas']['TeamMembershipCreate']
export type TeamMembershipUpdate = components['schemas']['TeamMembershipUpdate']
export type SetRoleRequest = components['schemas']['SetRoleRequest']

export async function listTeamMemberships(query?: { page?: number; per_page?: number }) {
  return client.GET('/team-memberships', { params: { query } })
}

export async function getTeamMembership(id: number) {
  return client.GET('/team-memberships/{id}', { params: { path: { id } } })
}

export async function createTeamMembership(body: TeamMembershipCreate) {
  return client.POST('/team-memberships', { body })
}

export async function updateTeamMembership(id: number, body: TeamMembershipUpdate) {
  return client.PATCH('/team-memberships/{id}', { params: { path: { id } }, body })
}

export async function deleteTeamMembership(id: number) {
  return client.DELETE('/team-memberships/{id}', { params: { path: { id } } })
}

export async function approveTeamMembership(id: number) {
  return client.POST('/team-memberships/{id}/approve', { params: { path: { id } } })
}

export async function rejectTeamMembership(id: number) {
  return client.POST('/team-memberships/{id}/reject', { params: { path: { id } } })
}

export async function acceptTeamMembership(id: number) {
  return client.POST('/team-memberships/{id}/accept', { params: { path: { id } } })
}

export async function declineTeamMembership(id: number) {
  return client.POST('/team-memberships/{id}/decline', { params: { path: { id } } })
}

export async function setTeamMembershipRole(id: number, body: SetRoleRequest) {
  return client.POST('/team-memberships/{id}/set-role', {
    params: { path: { id } },
    body,
  })
}
