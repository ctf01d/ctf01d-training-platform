import client from "./client";

export async function getGameScoreboard(gameId: number) {
  return client.GET("/games/{id}/scoreboard", {
    params: { path: { id: gameId } },
  });
}

export async function getGlobalScoreboard() {
  return client.GET("/scoreboard");
}
