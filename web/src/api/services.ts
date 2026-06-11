import client from './client'
import type { components } from './schema'

export type Service = components['schemas']['Service']
export type ServiceCreate = components['schemas']['ServiceCreate']
export type ServiceUpdate = components['schemas']['ServiceUpdate']

export async function listServices(query?: { page?: number; per_page?: number; public?: boolean; q?: string }) {
  return client.GET('/services', { params: { query } })
}

export async function getService(id: number) {
  return client.GET('/services/{id}', { params: { path: { id } } })
}

export async function createService(body: ServiceCreate) {
  return client.POST('/services', { body })
}

export async function updateService(id: number, body: ServiceUpdate) {
  return client.PATCH('/services/{id}', { params: { path: { id } }, body })
}

export async function deleteService(id: number) {
  return client.DELETE('/services/{id}', { params: { path: { id } } })
}

export async function toggleServicePublic(id: number) {
  return client.POST('/services/{id}/toggle-public', { params: { path: { id } } })
}

export async function checkServiceChecker(id: number) {
  return client.POST('/services/{id}/check-checker', { params: { path: { id } } })
}

export async function redownloadServiceArchives(id: number) {
  return client.POST('/services/{id}/redownload', { params: { path: { id } } })
}

export async function downloadServiceArchive(id: number, kind: 'service' | 'checker') {
  return client.GET('/services/{id}/download/{kind}', {
    params: { path: { id, kind } },
    parseAs: 'blob',
  })
}

export async function importServiceFromGithub(body: components['schemas']['GithubImportRequest']) {
  return client.POST('/services/import/github', { body })
}

export async function importServiceFromZip(formData: FormData) {
  const response = await fetch('/api/v1/services/import/zip', {
    method: 'POST',
    body: formData,
  })
  return response
}
