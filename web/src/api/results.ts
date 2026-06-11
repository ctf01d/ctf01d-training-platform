import client from './client'
import type { components } from './schema'

export type Result = components['schemas']['Result']
export type ResultCreate = components['schemas']['ResultCreate']
export type ResultUpdate = components['schemas']['ResultUpdate']

export async function listResults(query?: { game_id?: number; team_id?: number }) {
  return client.GET('/results', { params: { query } })
}

export async function getResult(id: number) {
  return client.GET('/results/{id}', { params: { path: { id } } })
}

export async function createResult(body: ResultCreate) {
  return client.POST('/results', { body })
}

export async function updateResult(id: number, body: ResultUpdate) {
  return client.PATCH('/results/{id}', { params: { path: { id } }, body })
}

export async function deleteResult(id: number) {
  return client.DELETE('/results/{id}', { params: { path: { id } } })
}
