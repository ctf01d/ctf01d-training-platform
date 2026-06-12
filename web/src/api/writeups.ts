import client from './client'
import type { components } from './schema'

export type Writeup = components['schemas']['Writeup']
export type WriteupCreate = components['schemas']['WriteupCreate']

export async function listWriteups(query?: { game_id?: number; team_id?: number }) {
  return client.GET('/writeups', { params: { query } })
}

export async function getWriteup(id: number) {
  return client.GET('/writeups/{id}', { params: { path: { id } } })
}

export async function createWriteup(body: WriteupCreate) {
  return client.POST('/writeups', { body })
}

export async function deleteWriteup(id: number) {
  return client.DELETE('/writeups/{id}', { params: { path: { id } } })
}
