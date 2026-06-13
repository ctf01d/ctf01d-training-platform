import client from "./client";
import type { components } from "./schema";

export type GameTeam = components["schemas"]["GameTeam"];
export type GameTeamCreate = components["schemas"]["GameTeamCreate"];
export type GameTeamUpdate = components["schemas"]["GameTeamUpdate"];

export async function createGameTeam(body: GameTeamCreate) {
  return client.POST("/game-teams", { body });
}

export async function updateGameTeam(id: number, body: GameTeamUpdate) {
  return client.PATCH("/game-teams/{id}", { params: { path: { id } }, body });
}

export async function deleteGameTeam(id: number) {
  return client.DELETE("/game-teams/{id}", { params: { path: { id } } });
}

export async function reorderGameTeams(
  gameId: number,
  items: { id: number; order: number }[],
) {
  return client.POST("/games/{id}/teams/reorder", {
    params: { path: { id: gameId } },
    body: { items },
  });
}
