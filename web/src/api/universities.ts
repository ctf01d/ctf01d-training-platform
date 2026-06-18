import client from "./client";
import type { components } from "./schema";

export type University = components["schemas"]["University"];
export type UniversityCreate = components["schemas"]["UniversityCreate"];
export type UniversityUpdate = components["schemas"]["UniversityUpdate"];

export async function listUniversities(query?: {
  page?: number;
  per_page?: number;
  q?: string;
}) {
  return client.GET("/universities", { params: { query } });
}

/**
 * Fetch every university across all pages. The list endpoint caps per_page at
 * 100, so callers needing the full set must paginate.
 */
export async function listAllUniversities(query?: {
  q?: string;
}): Promise<University[]> {
  const perPage = 100;
  const items: University[] = [];
  for (let page = 1; ; page++) {
    const { data } = await listUniversities({
      page,
      per_page: perPage,
      q: query?.q,
    });
    if (!data) break;
    items.push(...data.items);
    if (items.length >= data.pagination.total || data.items.length === 0) break;
  }
  return items;
}

export async function getUniversity(id: number) {
  return client.GET("/universities/{id}", { params: { path: { id } } });
}

export async function createUniversity(body: UniversityCreate) {
  return client.POST("/universities", { body });
}

export async function updateUniversity(id: number, body: UniversityUpdate) {
  return client.PATCH("/universities/{id}", { params: { path: { id } }, body });
}

export async function deleteUniversity(id: number) {
  return client.DELETE("/universities/{id}", { params: { path: { id } } });
}
