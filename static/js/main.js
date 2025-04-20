// static/js/main.js

import { Configuration, GamesApi } from './api'; // <-- путь к сгенерированному SDK

document.addEventListener('DOMContentLoaded', async () => {
  const config = new Configuration({
    basePath: '/', // или URL твоего API, если он на поддомене/порте
  });

  const api = new GamesApi(config);

  try {
    const response = await api.gamesList(); // или gamesListWithHttpInfo() если нужен response.headers
    const data = response.data;

    console.log('Games:', data);

    let html = '';
    if (!data || data.length === 0) {
      html = '<div class="alert alert-info">No games found.</div>';
    } else {
      html = '<ul class="list-group">';
      data.forEach((game) => {
        html += '<li class="list-group-item">';
        html += `<strong>#${game.id}</strong> — ${game.description}<br>`;
        html += `<small>Start: ${game.start_time}, End: ${game.end_time}</small>`;
        html += '</li>';
      });
      html += '</ul>';
    }

    document.getElementById('games-list').innerHTML = html;

  } catch (error) {
    console.error('API error:', error);
    document.getElementById('games-list').innerHTML =
      '<div class="alert alert-danger">Failed to load games</div>';
  }
});
