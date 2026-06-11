import client from './client'
import type { components } from './schema'

export type University = components['schemas']['University']
export type UniversityCreate = components['schemas']['UniversityCreate']
export type UniversityUpdate = components['schemas']['UniversityUpdate']

export async function listUniversities(query?: { page?: number; per_page?: number }) {
  return client.GET('/universities', { params: { query } })
}

export async function getUniversity(id: number) {
  return client.GET('/universities/{id}', { params: { path: { id } } })
}

export async function createUniversity(body: UniversityCreate) {
  return client.POST('/universities', { body })
}

export async function updateUniversity(id: number, body: UniversityUpdate) {
  return client.PATCH('/universities/{id}', { params: { path: { id } }, body })
}

export async function deleteUniversity(id: number) {
  return client.DELETE('/universities/{id}', { params: { path: { id } } })
}
