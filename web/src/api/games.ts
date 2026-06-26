import client from "./client";
import type { components } from "./schema";

export type Game = components["schemas"]["Game"];
export type GameCreate = components["schemas"]["GameCreate"];
export type GameUpdate = components["schemas"]["GameUpdate"];
export type GameServiceLink = components["schemas"]["GameServiceLink"];

export async function listGames(query?: {
  page?: number;
  per_page?: number;
  q?: string;
  published?: boolean;
}) {
  return client.GET("/games", { params: { query } });
}

/**
 * Fetch every game across all pages. The list endpoint caps per_page at 100,
 * so callers needing the full set must paginate.
 */
export async function listAllGames(query?: { q?: string }): Promise<Game[]> {
  const perPage = 100;
  const items: Game[] = [];
  for (let page = 1; ; page++) {
    const { data } = await listGames({ page, per_page: perPage, q: query?.q });
    if (!data) break;
    items.push(...data.items);
    if (items.length >= data.pagination.total || data.items.length === 0) break;
  }
  return items;
}

export async function getGame(id: number) {
  return client.GET("/games/{id}", { params: { path: { id } } });
}

export async function createGame(body: GameCreate) {
  return client.POST("/games", { body });
}

export async function updateGame(id: number, body: GameUpdate) {
  return client.PATCH("/games/{id}", { params: { path: { id } }, body });
}

export async function deleteGame(id: number) {
  return client.DELETE("/games/{id}", { params: { path: { id } } });
}

export async function listGameServices(id: number) {
  return client.GET("/games/{id}/services", { params: { path: { id } } });
}

export async function addGameService(
  id: number,
  serviceId: number,
  status?: string,
) {
  return client.POST("/games/{id}/services", {
    params: { path: { id } },
    body: { service_id: serviceId, status },
  });
}

export async function removeGameService(id: number, serviceId: number) {
  return client.DELETE("/games/{id}/services/{service_id}", {
    params: { path: { id, service_id: serviceId } },
  });
}

export async function setGameServiceStatus(
  id: number,
  serviceId: number,
  status: string,
) {
  return client.PATCH("/games/{id}/services/{service_id}", {
    params: { path: { id, service_id: serviceId } },
    body: { status },
  });
}

export async function publishGame(id: number) {
  return client.POST("/games/{id}/publish", { params: { path: { id } } });
}

export async function finalizeGame(id: number) {
  return client.POST("/games/{id}/finalize", { params: { path: { id } } });
}

export async function unfinalizeGame(id: number) {
  return client.POST("/games/{id}/unfinalize", { params: { path: { id } } });
}

export async function getCtf01dExportOptions(id: number) {
  return client.GET("/games/{id}/export/ctf01d/options", {
    params: { path: { id } },
  });
}

export async function exportCtf01d(
  id: number,
  body?: components["schemas"]["Ctf01dExportRequest"],
) {
  return client.POST("/games/{id}/export/ctf01d", {
    params: { path: { id } },
    body,
  });
}

export async function listGameTeams(id: number) {
  return client.GET("/games/{id}/teams", { params: { path: { id } } });
}
